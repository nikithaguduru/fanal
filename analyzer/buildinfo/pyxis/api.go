package pyxis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/xerrors"
)

const (
	pyxisAPI = "https://catalog.redhat.com/api/containers/v1/images/nvr/%s" +
		"?filter=parsed_data.labels=em=(name=='architecture'andvalue=='%s')"
)

type response struct {
	Data []struct {
		ContentSets []string `json:"content_sets"`
		CpeIDs      []string `json:"cpe_ids"`
	} `json:"data"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type Pyxis struct {
	baseURL string
}

type Option func(pyxis *Pyxis)

func WithURL(url string) Option {
	return func(pyxis *Pyxis) {
		pyxis.baseURL = url
	}
}

func NewPyxis(options ...Option) Pyxis {
	p := &Pyxis{
		baseURL: pyxisAPI,
	}
	for _, opt := range options {
		opt(p)
	}
	return *p
}

type mapping struct {
	Nvr         string   `json:"nvr"`
	Arch        string   `json:"arch"`
	ContentSets []string `json:"content_sets"`
	CpeIDs      []string `json:"cpe_ids"`
}

func (p Pyxis) FetchContentSets(nvr, arch string) ([]string, error) {
	url := fmt.Sprintf(p.baseURL, nvr, arch)
	resp, err := http.Get(url)
	if err != nil {
		return nil, xerrors.Errorf("HTTP error (%s): %w", url, err)
	}
	defer resp.Body.Close()

	var res response
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, xerrors.Errorf("JSON parse error: %w", err)
	}

	if len(res.Data) != 1 {
		return nil, xerrors.Errorf("the response must have only one block")
	}

	// TODO: For generating mapping
	f, err := os.Open("nvr-mapping.json")
	if err != nil {
		panic(err)
	}

	var m map[string]mapping
	if err = json.NewDecoder(f).Decode(&m); err != nil {
		panic(err)
	}
	f.Close()

	m[fmt.Sprintf("%s//%s", nvr, arch)] = mapping{
		Nvr:         nvr,
		Arch:        arch,
		ContentSets: res.Data[0].ContentSets,
		CpeIDs:      res.Data[0].CpeIDs,
	}

	f, err = os.Create("nvr-mapping.json")
	if err != nil {
		panic(err)
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err = enc.Encode(m); err != nil {
		panic(err)
	}
	f.Close()

	return res.Data[0].ContentSets, nil
}
