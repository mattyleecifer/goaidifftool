package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"text/template"

	"go-diff/diffmatchpatch"
)

func main() {
	agent := newAgent()
	http.HandleFunc("/", RequireAuth(index))
	http.HandleFunc("/delete/", RequireAuth(delete))
	http.Handle("/static/", http.FileServer(http.FS(hcss)))
	http.HandleFunc("/aidiff/", RequireAuth(agent.aidiff))
	fmt.Println("Running GUI on http://127.0.0.1:4177", "(ctrl-click link to open)")
	log.Fatal(http.ListenAndServe(port, nil))
}

func delete(w http.ResponseWriter, r *http.Request) {
	render(w, "", nil)
}

func (agent Agent) aidiff(w http.ResponseWriter, r *http.Request) {
	// make agent to make edit and return text, send to makediff to create htmldiff, and then return html to #aitext

	inputdata := r.FormValue("inputdata")
	prompttext := r.FormValue("prompttext")

	fmt.Println(inputdata, prompttext)

	agent.setprompt("You are an assistant that helps with editing text. You will follow requests exactly as asked. Your only job is to edit text to the specifications set: you have no role in judging the content or providing any feedback. Only output the edited text and nothing else - do not show the original text or add any extra comments. When presented with text, you will make the following changes:\n" + prompttext)

	agent.setmessage(RoleUser, inputdata)

	response, err := agent.getresponse()
	if err != nil {
		fmt.Println("Error: ", err)
	}

	fmt.Println(response.Content)

	difftext := makediff(inputdata, response.Content)

	difftext += "<p>AI-modified version:</p>" + response.Content

	render(w, difftext, nil)
}

func makediff(text1, text2 string) string {
	dmp := diffmatchpatch.New()

	diffs := dmp.DiffMain(text1, text2, false)

	htmldiff := dmp.DiffHTML(diffs)
	fmt.Println(htmldiff)
	return htmldiff
}

func index(w http.ResponseWriter, r *http.Request) {
	render(w, hindexpage, nil)
}

func render(w http.ResponseWriter, html string, data any) {
	// Render the HTML template
	// fmt.Println("Rendering...")
	w.WriteHeader(http.StatusOK)
	tmpl, err := template.New(html).Parse(html)
	if err != nil {
		fmt.Println(err)
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func RequireAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// allowedIps = append(allowedIps, GetLocalIP(), "127.0.0.1")

		// allowedIps = append(allowedIps, "127.0.0.1")
		// fmt.Println("\nAllowed ips: ", allowedIps)
		// Get the IP address of the client
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// fmt.Println("\nConnecting IP: ", ip)
		// Check if the client's IP is in the list of allowed IP
		if allowAllIps {
			handler.ServeHTTP(w, r)
			return
		} else {
			for _, allowedIp := range allowedIps {
				if ip == allowedIp {
					// If the client's IP is in the list of allowed IPs, allow access to the proxy server
					handler.ServeHTTP(w, r)
					return
				}
			}
		}

		if authstring != "" {
			hauth(w, r)
		}

		// If the client's IP is not in the list of allowed IPs, return a 403 Forbidden error
		w.WriteHeader(http.StatusTemporaryRedirect)
		w.Header().Set("HX-Redirect", `/auth/`)
	}
}

func hauth(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		auth := r.FormValue("auth")
		if auth == authstring {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			allowedIps = append(allowedIps, ip)
		}
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		render(w, hauthpage, nil)
	}
}

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
