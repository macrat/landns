package landns

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e HTTPError) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(e.StatusCode)
	fmt.Fprintln(w, e.Error())
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("; %d: %s", e.StatusCode, strings.ReplaceAll(e.Message, "\n", "\n;      "))
}

type httpHandler func(path string, body string) (string, *HTTPError)

func (hh httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		HTTPError{http.StatusBadRequest, "bad request"}.ServeHTTP(w, r)
		return
	}

	resp, e := hh(r.URL.Path, string(body))
	if e != nil {
		e.ServeHTTP(w, r)
		return
	}

	resp = strings.TrimRight(resp, "\n")
	if len(resp) != 0 {
		fmt.Fprintln(w, resp)
	}
}

type httpHandlerSet map[string]http.Handler

func (hhs httpHandlerSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := hhs[r.Method]; ok {
		h.ServeHTTP(w, r)
		return
	}

	HTTPError{http.StatusMethodNotAllowed, "method not allowed"}.ServeHTTP(w, r)
}

type DynamicAPI struct {
	resolver DynamicResolver
}

func NewDynamicAPI(resolver DynamicResolver) DynamicAPI {
	return DynamicAPI{resolver}
}

func (d DynamicAPI) GetAllRecords(path, req string) (string, *HTTPError) {
	records, err := d.resolver.Records()
	if err != nil {
		return "", &HTTPError{http.StatusInternalServerError, "internal server error"}
	}

	return records.String(), nil
}

func (d DynamicAPI) GetRecords(path, req string) (string, *HTTPError) {
	if path[len(path)-1] == '/' {
		return "", &HTTPError{http.StatusNotFound, "not found"}
	}

	items := strings.Split(path[len("/v1/suffix/"):], "/")
	rev := make([]string, len(items))
	for i := range items {
		rev[i] = items[len(items)-1-i]
	}
	domain := Domain(strings.Join(rev, "."))

	if err := domain.Validate(); err != nil || domain.String()[0] == '.' {
		return "", &HTTPError{http.StatusNotFound, "not found"}
	}

	records, err := d.resolver.SearchRecords(domain)
	if err != nil {
		return "", &HTTPError{http.StatusInternalServerError, "internal server error"}
	}

	return records.String(), nil
}

func (d DynamicAPI) setRecords(rs DynamicRecordSet) (string, *HTTPError) {
	if err := d.resolver.SetRecords(rs); err != nil {
		return "", &HTTPError{http.StatusInternalServerError, "internal server error"}
	}

	add := 0
	del := 0
	for _, r := range rs {
		if r.Disabled {
			del++
		} else {
			add++
		}
	}

	return fmt.Sprintf("; 200: add:%d delete:%d", add, del), nil
}

func (d DynamicAPI) PostRecords(path, req string) (string, *HTTPError) {
	rs, err := NewDynamicRecordSet(req)
	if err != nil {
		return "", &HTTPError{http.StatusBadRequest, err.Error()}
	}

	return d.setRecords(rs)
}

func (d DynamicAPI) DeleteRecords(path, req string) (string, *HTTPError) {
	rs, err := NewDynamicRecordSet(req)
	if err != nil {
		return "", &HTTPError{http.StatusBadRequest, err.Error()}
	}

	for i := range rs {
		rs[i].Disabled = !rs[i].Disabled
	}

	return d.setRecords(rs)
}

func (d DynamicAPI) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/v1", httpHandlerSet{
		"GET":    httpHandler(d.GetAllRecords),
		"POST":   httpHandler(d.PostRecords),
		"DELETE": httpHandler(d.DeleteRecords),
	})
	mux.Handle("/v1/suffix", httpHandlerSet{
		"GET": httpHandler(d.GetAllRecords),
	})
	mux.Handle("/v1/suffix/", httpHandlerSet{
		"GET": httpHandler(d.GetRecords),
	})

	return mux
}
