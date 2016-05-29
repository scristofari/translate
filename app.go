package translate

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"

	"google.golang.org/appengine"
	"google.golang.org/appengine/memcache"
	"google.golang.org/appengine/urlfetch"
)

var (
	apiKEy           = "your-key"
	translateURLRoot = "https://www.googleapis.com/language/translate/v2"
)

func init() {
	gob.Register(googleTranslateAPIResponse{})

	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/translate", translateHandler).Methods("GET")
	http.Handle("/", r)
}

type googleTranslateAPIResponse struct {
	Data struct {
		Translations []struct {
			TranslatedText string `json:"translatedText"`
		} `json:"translations"`
	} `json:"data"`
}

type appRequest struct {
	Message string `schema:"q"`
	Source  string `schema:"source"`
	Target  string `schema:"target"`
}

type appResponse struct {
	Result string `json:"result"`
}

func (ar *appRequest) key() string {
	return fmt.Sprintf("%s-%s-%s", url.QueryEscape(ar.Message), ar.Source, ar.Target)
}

func translateHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	arq := new(appRequest)
	if err := bindForm(r, arq); err != nil {
		http.Error(w, "Error : Failed to bind request", http.StatusInternalServerError)
		return
	}

	gta := new(googleTranslateAPIResponse)
	_, err := memcache.Gob.Get(ctx, arq.key(), gta)

	if err == memcache.ErrCacheMiss {

		u := fmt.Sprintf(translateURLRoot+"?key=%s&q=%s&source=%s&target=%s", apiKEy, url.QueryEscape(arq.Message), arq.Source, arq.Target)

		client := urlfetch.Client(ctx)
		resp, err := client.Get(u)

		if err != nil {
			http.Error(w, "Error : Failed to request the google api", http.StatusInternalServerError)
			return
		}

		if resp.StatusCode != http.StatusOK {
			http.Error(w, fmt.Sprintf("Error : Api call failed (status:%d)", resp.StatusCode), http.StatusInternalServerError)
			return
		}

		if err := decodeJSON(resp, gta); err != nil {
			http.Error(w, fmt.Sprintf("Error : Failed to bind google api JSON (%s)", err.Error()), http.StatusInternalServerError)
			return
		}

		// 1 day
		var expiration = time.Duration(86400) * time.Second
		nt := &memcache.Item{
			Key:        arq.key(),
			Object:     gta,
			Expiration: expiration,
		}

		if err := memcache.Gob.Set(ctx, nt); err != nil {
			http.Error(w, fmt.Sprintf("Error : Failed to save in memcache", err.Error()), http.StatusInternalServerError)
			return
		}

	} else if err != nil {
		http.Error(w, "Error : Failed to request the google api", http.StatusInternalServerError)
		return
	}

	ap := new(appResponse)
	ap.Result = gta.Data.Translations[0].TranslatedText

	writeJSON(w, ap, http.StatusOK)
}
