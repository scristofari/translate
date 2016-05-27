package translate

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/schema"
)

// Deserialize json to struct
func decodeJSON(r *http.Response, obj interface{}) error {
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	if err := decoder.Decode(obj); err != nil {
		return err
	}
	return nil
}

// Bind form base to struct
// Gorilla schema
func bindForm(r *http.Request, obj interface{}) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	decoder := schema.NewDecoder()
	return decoder.Decode(obj, r.Form)
}

// Serialize struct to json
// Write the correct status / content-type
func writeJSON(w http.ResponseWriter, obj interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.Encode(obj)
}
