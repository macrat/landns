package landns

import (
	"log"
	"net/http"

	"github.com/labstack/echo"
	"github.com/miekg/dns"
)

var (
	BadRqeustError = echo.NewHTTPError(http.StatusBadRequest, "invalid request")
)

type DynamicAPI struct {
	resolver DynamicResolver
}

func NewDynamicAPI(resolver DynamicResolver) DynamicAPI {
	return DynamicAPI{resolver}
}

func (d DynamicAPI) GetAddresses(c echo.Context) error {
	addrs, err := d.resolver.GetAddresses()
	if err != nil {
		log.Printf("dynamic-zone: %s", err)
		return err
	}

	return c.JSON(http.StatusOK, addrs)
}

func (d DynamicAPI) UpdateAddresses(c echo.Context) error {
	var request AddressesConfig

	if err := c.Bind(&request); err != nil {
		return BadRqeustError
	} else if err = request.Validate(); err != nil {
		return BadRqeustError
	}

	if err := d.resolver.UpdateAddresses(request); err != nil {
		log.Printf("dynamic-zone: %s", err)
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (d DynamicAPI) ResolveAddress(c echo.Context) error {
	name := Domain(c.Param("name"))

	if err := name.Validate(); err != nil {
		return BadRqeustError
	}

	resp := make([]Record, 0, 10)
	writer := NewResponseCallback(func(r Record) error {
		resp = append(resp, r)
		return nil
	})

	if err := d.resolver.Resolve(writer, NewRequest(name.String(), dns.TypeA, false)); err != nil {
		log.Printf("dynamic-zone: %s", err)
		return nil
	}
	if err := d.resolver.Resolve(writer, NewRequest(name.String(), dns.TypeAAAA, false)); err != nil {
		log.Printf("dynamic-zone: %s", err)
		return nil
	}

	return c.JSON(http.StatusOK, resp)
}

func (d DynamicAPI) GetCnames(c echo.Context) error {
	cnames, err := d.resolver.GetCnames()
	if err != nil {
		log.Printf("dynamic-zone: %s", err)
		return err
	}

	return c.JSON(http.StatusOK, cnames)
}

func (d DynamicAPI) UpdateCnames(c echo.Context) error {
	var request CnamesConfig

	if err := c.Bind(&request); err != nil {
		return BadRqeustError
	} else if err = request.Validate(); err != nil {
		return BadRqeustError
	}

	if err := d.resolver.UpdateCnames(request); err != nil {
		log.Printf("dynamic-zone: %s", err)
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (d DynamicAPI) ResolveCname(c echo.Context) error {
	name := Domain(c.Param("name"))

	if err := name.Validate(); err != nil {
		return BadRqeustError
	}

	resp := make([]Record, 0, 10)
	writer := NewResponseCallback(func(r Record) error {
		resp = append(resp, r)
		return nil
	})
	if err := d.resolver.Resolve(writer, NewRequest(name.String(), dns.TypeCNAME, false)); err != nil {
		log.Printf("dynamic-zone: %s", err)
		return nil
	}

	return c.JSON(http.StatusOK, resp)
}

func (d DynamicAPI) GetTexts(c echo.Context) error {
	texts, err := d.resolver.GetTexts()
	if err != nil {
		log.Printf("dynamic-zone: %s", err)
		return err
	}

	return c.JSON(http.StatusOK, texts)
}

func (d DynamicAPI) UpdateTexts(c echo.Context) error {
	var request TextsConfig

	if err := c.Bind(&request); err != nil {
		return BadRqeustError
	} else if err = request.Validate(); err != nil {
		return BadRqeustError
	}

	if err := d.resolver.UpdateTexts(request); err != nil {
		log.Printf("dynamic-zone: %s", err)
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (d DynamicAPI) ResolveText(c echo.Context) error {
	name := Domain(c.Param("name"))

	if err := name.Validate(); err != nil {
		return BadRqeustError
	}

	resp := make([]Record, 0, 10)
	writer := NewResponseCallback(func(r Record) error {
		resp = append(resp, r)
		return nil
	})
	if err := d.resolver.Resolve(writer, NewRequest(name.String(), dns.TypeTXT, false)); err != nil {
		log.Printf("dynamic-zone: %s", err)
		return nil
	}

	return c.JSON(http.StatusOK, resp)
}

func (d DynamicAPI) GetServices(c echo.Context) error {
	services, err := d.resolver.GetServices()
	if err != nil {
		log.Printf("dynamic-zone: %s", err)
		return err
	}

	return c.JSON(http.StatusOK, services)
}

func (d DynamicAPI) UpdateServices(c echo.Context) error {
	var request ServicesConfig

	if err := c.Bind(&request); err != nil {
		return BadRqeustError
	} else if err = request.Validate(); err != nil {
		return BadRqeustError
	}

	if err := d.resolver.UpdateServices(request); err != nil {
		log.Printf("dynamic-zone: %s", err)
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (d DynamicAPI) ResolveService(c echo.Context) error {
	name := Domain(c.Param("name"))

	if err := name.Validate(); err != nil {
		return BadRqeustError
	}

	resp := make([]Record, 0, 10)
	writer := NewResponseCallback(func(r Record) error {
		resp = append(resp, r)
		return nil
	})
	if err := d.resolver.Resolve(writer, NewRequest(name.String(), dns.TypeSRV, false)); err != nil {
		log.Printf("dynamic-zone: %s", err)
		return nil
	}

	return c.JSON(http.StatusOK, resp)
}

func (d DynamicAPI) GetAllRecords(c echo.Context) (err error) {
	resp := ResolverConfig{}

	resp.Addresses, err = d.resolver.GetAddresses()
	if err != nil {
		log.Print(err.Error())
		return
	}

	resp.Cnames, err = d.resolver.GetCnames()
	if err != nil {
		log.Print(err.Error())
		return
	}

	resp.Texts, err = d.resolver.GetTexts()
	if err != nil {
		log.Print(err.Error())
		return
	}

	resp.Services, err = d.resolver.GetServices()
	if err != nil {
		log.Print(err.Error())
		return
	}

	return c.JSON(http.StatusOK, resp)
}

func (d DynamicAPI) Handler() http.Handler {
	e := echo.New()

	e.GET("/v1/record/address", d.GetAddresses)
	e.POST("/v1/record/address", d.UpdateAddresses)
	e.GET("/v1/record/address/:name", d.ResolveAddress)

	e.GET("/v1/record/text", d.GetTexts)
	e.POST("/v1/record/text", d.UpdateTexts)
	e.GET("/v1/record/text/:name", d.ResolveText)

	e.GET("/v1/record/cname", d.GetCnames)
	e.POST("/v1/record/cname", d.UpdateCnames)
	e.GET("/v1/record/cname/:name", d.ResolveCname)

	e.GET("/v1/record/service", d.GetServices)
	e.POST("/v1/record/service", d.UpdateServices)
	e.GET("/v1/record/service/:name", d.ResolveService)

	e.GET("/v1/record", d.GetAllRecords)

	return e
}
