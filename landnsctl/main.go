package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/go-yaml/yaml"
	"github.com/macrat/landns/lib-landns"
)

var (
	app      = kingpin.New("landnsctl", "A command-line client for Landns.")
	endpoint = kingpin.Flag("endpoint", "The endpoint of Landns API.").Default("http://localhost:9353/api/v1/").URL()
	ttl      = kingpin.Flag("ttl", "TTL for set record.").Default("3600").Uint()
)

func forceURL(raw_url string) *url.URL {
	if u, err := url.Parse(raw_url); err != nil {
		panic(err.Error())
	} else {
		return u
	}
}

func Get(path *url.URL) error {
	resp, err := http.Get((*endpoint).ResolveReference(path).String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var buf interface{}
	if err = json.Unmarshal(body, &buf); err != nil {
		return err
	}

	output, err := yaml.Marshal(buf)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK {
		if string(output) != "[]\n" {
			fmt.Print(string(output))
		}
		return nil
	}

	fmt.Print(string(output))
	os.Exit(1)
	return nil
}

func Post(path *url.URL, data interface{}) error {
	msg, err := json.Marshal(data)
	if err != nil {
		return err
	}

	resp, err := http.Post((*endpoint).ResolveReference(path).String(), "application/json", bytes.NewBuffer(msg))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if len(body) == 0 {
		return nil
	}

	var buf interface{}
	if err = json.Unmarshal(body, &buf); err != nil {
		return err
	}

	output, err := yaml.Marshal(buf)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK {
		if string(output) != "[]\n" {
			fmt.Print(string(output))
		}
		return nil
	} else if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	fmt.Print(string(output))
	os.Exit(1)
	return nil
}

func AddressCommand(app *kingpin.Application) {
	cmd := kingpin.Command("addr", "Set or get A/AAAA record.")
	name := cmd.Arg("name", "Domain name to operate.").Required().String()
	addrs := cmd.Flag("set", "Addresses to set to the specified domain.").Short('s').Strings()
	unset := cmd.Flag("unset", "Unset records.").Short('u').Bool()

	cmd.Action(func(ctx *kingpin.ParseContext) error {
		domain := landns.Domain(*name).Normalized()

		if err := domain.Validate(); err != nil {
			return err
		}

		if *unset {
			return Post(forceURL("record/address"), landns.AddressesConfig{
				domain: []landns.AddressRecordConfig{},
			})
		} else if len(*addrs) > 0 {
			records := []landns.AddressRecordConfig{}

			for _, a := range *addrs {
				ip := net.ParseIP(a)

				records = append(records, landns.AddressRecordConfig{
					TTL:     ttl,
					Address: ip,
				})
			}

			return Post(forceURL("record/address"), landns.AddressesConfig{
				domain: records,
			})
		} else {
			return Get(forceURL("record/address/" + domain.String()))
		}
	})
}

func CnameCommand(app *kingpin.Application) {
	cmd := kingpin.Command("cname", "Set or get CNAME record.")
	name := cmd.Arg("name", "Domain name to operate.").Required().String()
	targets := cmd.Flag("set", "Target domains to set to the specified domain.").Short('s').Strings()
	unset := cmd.Flag("unset", "Unset records.").Short('u').Bool()

	cmd.Action(func(ctx *kingpin.ParseContext) error {
		domain := landns.Domain(*name).Normalized()

		if err := domain.Validate(); err != nil {
			return err
		}

		if *unset {
			return Post(forceURL("record/cname"), landns.CnamesConfig{
				domain: []landns.CnameRecordConfig{},
			})
		} else if len(*targets) > 0 {
			records := []landns.CnameRecordConfig{}

			for _, t := range *targets {
				t_ := landns.Domain(t).Normalized()

				if err := t_.Validate(); err != nil {
					return err
				}

				records = append(records, landns.CnameRecordConfig{
					TTL:    ttl,
					Target: t_,
				})
			}

			return Post(forceURL("record/cname"), landns.CnamesConfig{
				domain: records,
			})
		} else {
			return Get(forceURL("record/cname/" + domain.String()))
		}
	})
}

func TextCommand(app *kingpin.Application) {
	cmd := kingpin.Command("text", "Set or get TXT record.")
	name := cmd.Arg("name", "Domain name to operate.").Required().String()
	texts := cmd.Flag("set", "Texts to set to the specified domain.").Short('s').Strings()
	unset := cmd.Flag("unset", "Unset records.").Short('u').Bool()

	cmd.Action(func(ctx *kingpin.ParseContext) error {
		domain := landns.Domain(*name).Normalized()

		if err := domain.Validate(); err != nil {
			return err
		}

		if *unset {
			return Post(forceURL("record/text"), landns.TextsConfig{
				domain: []landns.TxtRecordConfig{},
			})
		} else if len(*texts) > 0 {
			records := []landns.TxtRecordConfig{}

			for _, t := range *texts {
				records = append(records, landns.TxtRecordConfig{
					TTL:  ttl,
					Text: t,
				})
			}

			return Post(forceURL("record/text"), landns.TextsConfig{
				domain: records,
			})
		} else {
			return Get(forceURL("record/text/" + domain.String()))
		}
	})
}

func ServiceCommand(app *kingpin.Application) {
	cmd := kingpin.Command("service", "Set or get SRV record. (NOT IMPLEMENTED YET)")
	cmd.Arg("name", "Domain name to operate.").Required().String()

	cmd.Action(func(ctx *kingpin.ParseContext) error {
		return fmt.Errorf("not implemented yet")
	})
}

func init() {
	AddressCommand(app)
	CnameCommand(app)
	TextCommand(app)
	ServiceCommand(app)
}

func main() {
	kingpin.Parse()
}
