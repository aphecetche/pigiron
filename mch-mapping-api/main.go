package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"

	"github.com/mrrtf/pigiron/mapping"
	v2 "github.com/mrrtf/pigiron/mch-mapping-api/v2"
	"github.com/spf13/viper"

	// must include the specific implementation package of the mapping
	_ "github.com/mrrtf/pigiron/mapping/impl4"
)

var (
	ErrMissingDeId         = errors.New("Specifying a detection element id (deid=[number]) is required")
	ErrMissingBending      = errors.New("Specifying a bending plane (bending=true or bending=false) is required")
	ErrDeIdShouldBeInteger = errors.New("deid should be an integer")
	ErrInvalidBending      = errors.New("bending should be true or false")
	ErrInvalidDeId         error
	validdeids             []int
)

func init() {
	mapping.ForEachDetectionElement(func(i mapping.DEID) {
		validdeids = append(validdeids, int(i))
	})
	sort.Ints(validdeids)
	s, _ := json.Marshal(validdeids)
	ErrInvalidDeId = errors.New("Invalid deid. Possible values are :" + string(s))
}

type Bending struct {
	present bool
	value   bool
}

// getDeIdBending decode the query part of the url, expecting it to be
// of the form : deid=[number]&bending=[true|false].
// the bending part is optional.
func getDeIdBending(u *url.URL) (int, Bending, error) {
	q := u.Query()
	de, ok := q["deid"]
	if !ok {
		return -1, Bending{}, ErrMissingDeId
	}
	deid, err := strconv.Atoi(de[0])
	if err != nil {
		return -1, Bending{}, ErrDeIdShouldBeInteger
	}
	l := sort.SearchInts(validdeids, deid)
	if l >= len(validdeids) || validdeids[l] != deid {
		return -1, Bending{}, ErrInvalidDeId
	}
	_, ok = q["bending"]
	if ok {
		b, err := strconv.ParseBool(q["bending"][0])
		if err != nil {
			return -1, Bending{}, ErrInvalidBending
		}
		return deid, Bending{present: true, value: b}, nil
	}
	return deid, Bending{present: false}, nil
}

func makeHandler(fn func(w http.ResponseWriter, r *http.Request, deid int, bending bool), isBendingRequired bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-type", "application/json")
		deid, bending, err := getDeIdBending(r.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if isBendingRequired && !bending.present {
			http.Error(w, ErrMissingBending.Error(), http.StatusBadRequest)
			return
		}
		fn(w, r, deid, bending.value)
	}
}

func dualSampas(w http.ResponseWriter, r *http.Request, deid int, bending bool) {
	cseg := mapping.NewCathodeSegmentation(mapping.DEID(deid), bending)
	jsonDualSampas(w, cseg, bending)
}

func deGeo(w http.ResponseWriter, r *http.Request, deid int, bending bool) {
	cseg := mapping.NewCathodeSegmentation(mapping.DEID(deid), bending)
	jsonDEGeo(w, cseg, bending)
}

func handler() http.Handler {
	r := http.NewServeMux()
	r.HandleFunc("/", usage())
	bendingIsRequired := true
	r.HandleFunc("/dualsampas", makeHandler(dualSampas, bendingIsRequired))
	r.HandleFunc("/v2/dualsampas", makeHandler(v2.DualSampas, bendingIsRequired))
	r.HandleFunc("/degeo", makeHandler(deGeo, bendingIsRequired))
	return r
}

func main() {
	viper.SetEnvPrefix("MCH")
	viper.BindEnv("MAPPING_API_PORT")
	viper.SetDefault("MAPPING_API_PORT", 8080)
	port := viper.GetInt("MAPPING_API_PORT")
	fmt.Println("Started server to listen on port", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), handler()); err != nil {
		panic(err)
	}
}
