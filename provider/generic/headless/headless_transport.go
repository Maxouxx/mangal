package headless

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/gson"
	"net/http"
	"runtime"
	"strings"
	"sync"
)

var (
	transportInstance *Transport
	once              sync.Once
)

type Transport struct {
	browser *rod.Browser
}

func GetTransportSingleton() *Transport {
	once.Do(func() {
		u := launcher.New().Leakless(runtime.GOOS == "linux").Revision(1131003).Set(flags.Headless, "new").MustLaunch()
		browser := rod.New().ControlURL(u).MustConnect()
		transportInstance = &Transport{
			browser: browser,
		}
	})
	return transportInstance
}

func (t Transport) Close() error {
	return t.browser.Close()
}

// SetExtraHeaders whether to always send extra HTTP headers with the requests from this page.
func setExtraHeaders(p *rod.Page, headers http.Header) (func(), error) {
	gsonHeaders := proto.NetworkHeaders{}

	for key, values := range headers {
		gsonHeaders[key] = gson.New(strings.Join(values, ";"))
	}

	return p.EnableDomain(&proto.NetworkEnable{}), proto.NetworkSetExtraHTTPHeaders{Headers: gsonHeaders}.Call(p)
}

func (t Transport) RoundTrip(request *http.Request) (*http.Response, error) {

	page, err := t.browser.Context(request.Context()).Page(proto.TargetCreateTarget{URL: ""})

	if err != nil {
		return nil, err
	}

	_, err = page.SetExtraHeaders([]string{"Referer", request.Header.Get("Referer")})
	if err != nil {
		return nil, err
	}

	err = page.Navigate(request.URL.String())
	if err != nil {
		return nil, err
	}
	err = page.WaitLoad()

	if err != nil {
		return nil, err
	}
	return &http.Response{Body: newPageReader(page),
		StatusCode: 200,
		Header:     map[string][]string{"Content-Type": {"text/html"}},
	}, nil
}
