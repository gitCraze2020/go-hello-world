package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strings"
    "time"
)
func main() {
    mw := multiWeatherProvider{
        openWeatherMap{},
        // wundergroundAPIKey=4f24da5ff5f64027a4da5ff5f6b027a2
        weatherUnderground{apiKey: "5fd04433b59c1c47106bb4cac912498b"},
    }

    http.HandleFunc("/weather/", func(w http.ResponseWriter, r *http.Request) {
        begin := time.Now()
        city := strings.SplitN(r.URL.Path, "/", 3)[2]

        temp, err := mw.temperature(city)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "city:": city,
            "temp:": temp,
            "took:": time.Since(begin).String(),
        })
    })
    http.HandleFunc("/hello", hello)
    http.ListenAndServe(":8080", nil)

}
func say(s string) {
    for i := 0; i < 5; i++ {
        time.Sleep(100 * time.Millisecond)
        fmt.Println(s)
    }
}
func hello(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("hello!"))
}
type weatherData struct {
    Name string `json:"name"`
    Main struct {
        Kelvin float64 `json:"temp"`
    } `json:"main"`
}
type weatherProvider interface {
    temperature(city string) (float64, error) // in Kelvin, naturally
}
type openWeatherMap struct{}

//func query(city string) (weatherData, error) {
func (w openWeatherMap) temperature(city string) (float64, error) {
    var weatherAPIKey string = "5fd04433b59c1c47106bb4cac912498b"
    var url = "http://api.openweathermap.org/data/2.5/weather?APPID=" + weatherAPIKey + "&q=" + city
    resp, err := http.Get(url)
    if err != nil {
        //return weatherData{}, err
        return 0, err
    }

    defer resp.Body.Close()

    //var d weatherData
    var d struct {
        Main struct {
            Kelvin float64 `json:"temp"`
        } `json:"main"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, err
        //return weatherData{}, err
    }

    log.Printf("openWeatherMap: %s: %.2f", city, d.Main.Kelvin)
    return d.Main.Kelvin, nil
}
type weatherUnderground struct {
    apiKey string
}
func (w weatherUnderground) temperature(city string) (float64, error) {

    // NOTE weather underground api has changed, v3 is complex, not worth the effort for this exercise
    // simulation substituting openweathermap
    //
    //var url = "http://api.wunderground.com/api/" + w.apiKey + "/conditions/q/" + city + ".json"
    //var url = "https://api.weather.com/v3/location/search?query=" + city + "&locationType=locid&language=en-US&format=json&apiKey=" + w.apiKey + ".json"
    //var url = "https://api.weather.com/v3/location/search?query=" + city + "&locationType=locid&language=en-US&format=json&apiKey=" + w.apiKey


    var url = "http://api.openweathermap.org/data/2.5/weather?APPID=" + w.apiKey + "&q=" + city
    resp, err := http.Get(url)
    //resp, err := http.Get("http://api.wunderground.com/api/" + w.apiKey + "/conditions/q/" + city + ".json")
    if err != nil {
        return 0, err
    }

    defer resp.Body.Close()

    //var d struct {
    //    Observation struct {
    //        Celsius float64 `json:"temp_c"`
    //    } `json:"current_observation"`
    //}
    var d struct {
        Main struct {
            Kelvin float64 `json:"temp"`
        } `json:"main"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, err
    }

    //kelvin := d.Observation.Celsius + 273.15
    kelvin := d.Main.Kelvin
    log.Printf("weatherUnderground: %s: %.2f", city, kelvin)
    return kelvin, nil
}
func temperature(city string, providers ...weatherProvider) (float64, error) {
    sum := 0.0

    for _, provider := range providers {
        k, err := provider.temperature(city)
        if err != nil {
            return 0, err
        }

        sum += k
    }

    return sum / float64(len(providers)), nil
}

type multiWeatherProvider []weatherProvider

func (w multiWeatherProvider) temperature(city string) (float64, error) {
    // Make a channel for temperatures, and a channel for errors.
    // Each provider will push a value into only one.
    temps := make(chan float64, len(w))
    errs := make(chan error, len(w))

    // For each provider, spawn a goroutine with an anonymous function.
    // That function will invoke the temperature method, and forward the response.
    for _, provider := range w {
        go func(p weatherProvider) {
            k, err := p.temperature(city)
            if err != nil {
                errs <- err
                return
            }
            temps <- k
        }(provider)
    }

    sum := 0.0

    // Collect a temperature or an error from each provider.
    for i := 0; i < len(w); i++ {
        select {
        case temp := <-temps:
            sum += temp
        case err := <-errs:
            return 0, err
        }
    }

    //for _, provider := range w {
    //    k, err := provider.temperature(city)
    //    if err != nil {
    //        return 0, err
    //    }
    //    sum += k
    //}

    return sum / float64(len(w)), nil
}

