package web

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/ruudk/dead-code-analyzer/server/collector"
	"github.com/wcharczuk/go-chart"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Server struct {
	startTime time.Time
	collector *collector.Collector
	srv       *http.Server
}

func (w *Server) ListenAndServe() error {
	return w.srv.ListenAndServe()
}

func NewWebServer(collector *collector.Collector, port int) *Server {
	web := &Server{
		startTime: time.Now(),
		collector: collector,
	}

	web.srv = &http.Server{
		Handler: web.newRouter(),
		Addr:    fmt.Sprintf(":%d", port),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}

	return web
}

func (w *Server) newRouter() *mux.Router {
	router := mux.NewRouter()
	router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	router.HandleFunc("/ready-check", w.readyCheck).Methods("GET")
	router.HandleFunc("/health-check", w.healthCheck).Methods("GET")
	router.HandleFunc("/reset", w.reset).Methods("GET")
	router.HandleFunc("/reset", w.resetPost).Methods("POST")
	router.HandleFunc("/remove", w.remove).Methods("GET")
	router.HandleFunc("/remove", w.removePost).Methods("POST")
	router.HandleFunc("/dead", w.dead).Methods("GET")
	router.HandleFunc("/active", w.active).Methods("GET")
	router.HandleFunc("/chart", w.chart).Methods("GET")
	router.HandleFunc("/", w.index).Methods("GET")

	return router
}

func (w *Server) readyCheck(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func (w *Server) healthCheck(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func (w *Server) index(writer http.ResponseWriter, request *http.Request) {
	if len(w.collector.Storage.AutoLoaded) == 0 {
		http.Redirect(writer, request, "/reset", 302)

		return
	}

	t, err := template.ParseFiles("web/templates/layout.html", "web/templates/index.html")
	if err != nil {
		panic(err)
	}

	type Data struct {
		CurrentPage string
		Since       string
	}

	err = t.ExecuteTemplate(writer, "layout", Data{
		"home",
		w.collector.Storage.Since.Format(time.RFC822),
	})
	if err != nil {
		fmt.Printf("error executing template: %s\n", err)
	}
}

func (w *Server) active(writer http.ResponseWriter, request *http.Request) {
	type ClassCount struct {
		Class string
		Count int
	}

	type Data struct {
		CurrentPage string
		Since       string
		Active      []ClassCount
		ActiveCount int
	}

	data := Data{
		CurrentPage: "active",
		Since:       w.collector.Storage.Since.Format(time.RFC822),
		ActiveCount: 0,
	}

	w.collector.Mutex.RLock()
	for k, v := range w.collector.Storage.AutoLoaded {
		if v != 0 {
			data.Active = append(data.Active, ClassCount{k, v})
			data.ActiveCount++
		}
	}
	w.collector.Mutex.RUnlock()

	sort.Slice(data.Active, func(i, j int) bool {
		return data.Active[i].Count > data.Active[j].Count
	})

	t, err := template.ParseFiles("web/templates/layout.html", "web/templates/active.html")
	if err != nil {
		panic(err)
	}

	err = t.ExecuteTemplate(writer, "layout", data)
	if err != nil {
		fmt.Printf("error executing template: %s\n", err)
	}
}

func (w *Server) dead(writer http.ResponseWriter, request *http.Request) {
	type ClassCount struct {
		Class string
		Count int
	}

	type Data struct {
		CurrentPage string
		Since       string
		Dead        []string
		DeadCount   int
	}

	data := Data{
		CurrentPage: "dead",
		Since:       w.collector.Storage.Since.Format(time.RFC822),
		Dead: []string{},
		DeadCount:   0,
	}

	w.collector.Mutex.RLock()
	for k, v := range w.collector.Storage.AutoLoaded {
		if v == 0 {
			data.Dead = append(data.Dead, k)
			data.DeadCount++
		}
	}
	w.collector.Mutex.RUnlock()

	sort.Strings(data.Dead)

	t, err := template.ParseFiles("web/templates/layout.html", "web/templates/dead.html")
	if err != nil {
		panic(err)
	}

	err = t.ExecuteTemplate(writer, "layout", data)
	if err != nil {
		fmt.Printf("error executing template: %s\n", err)
	}
}

func (w *Server) chart(writer http.ResponseWriter, request *http.Request) {
	var active = 0
	var dead = 0

	w.collector.Mutex.RLock()
	for _, i := range w.collector.Storage.AutoLoaded {
		if i == 0 {
			dead++
		} else {
			active++
		}
	}
	w.collector.Mutex.RUnlock()

	pie := chart.PieChart{
		Width:  512,
		Height: 512,
		Values: []chart.Value{
			{
				Style: chart.Style{FillColor: chart.ColorGreen},
				Value: float64(active),
				Label: fmt.Sprintf("Active (%d)", active),
			},
			{
				Style: chart.Style{FillColor: chart.ColorRed},
				Value: float64(dead),
				Label: fmt.Sprintf("Dead (%d)", dead),
			},
		},
	}

	writer.Header().Set("Content-Type", "image/png")
	err := pie.Render(chart.PNG, writer)
	if err != nil {
		fmt.Printf("Error rendering pie chart: %v\n", err)
	}
}

func (w *Server) reset(writer http.ResponseWriter, request *http.Request) {
	t, err := template.ParseFiles("web/templates/layout.html", "web/templates/reset.html")
	if err != nil {
		panic(err)
	}

	type Data struct {
		CurrentPage string
	}

	err = t.ExecuteTemplate(writer, "layout", Data{"reset"})
	if err != nil {
		fmt.Printf("error executing template: %s\n", err)
	}
}

func (w *Server) resetPost(writer http.ResponseWriter, request *http.Request) {
	request.ParseForm()

	classes := strings.Split(strings.TrimSpace(request.Form.Get("classes")), "\n")
	w.collector.Reset()
	for _, c := range classes {
		w.collector.IncrementAutoLoadedClass(strings.TrimSpace(c), 0)
	}

	http.Redirect(writer, request, "/", 302)
}

func (w *Server) remove(writer http.ResponseWriter, request *http.Request) {
	t, err := template.ParseFiles("web/templates/layout.html", "web/templates/remove.html")
	if err != nil {
		panic(err)
	}

	type Data struct {
		CurrentPage string
	}

	err = t.ExecuteTemplate(writer, "layout", Data{"remove"})
	if err != nil {
		fmt.Printf("error executing template: %s\n", err)
	}
}

func (w *Server) removePost(writer http.ResponseWriter, request *http.Request) {
	request.ParseForm()

	classes := strings.Split(strings.TrimSpace(request.Form.Get("classes")), "\n")

	for _, c := range classes {
		w.collector.RemoveClass(strings.TrimSpace(c))
	}

	http.Redirect(writer, request, "/", 302)
}
