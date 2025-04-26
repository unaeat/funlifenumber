package main

import (
	"net/http"
)

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/web.html")
	})

	http.HandleFunc("/generate-pdf", handler)
	http.ListenAndServe(":5566", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.FormValue("passport_name")
	nickName := r.FormValue("nickname")
	birthday := r.FormValue("birthday_time")

	pdfData, errCode := generate(name, nickName, birthday)
	if errCode != http.StatusOK {
		http.Error(w, "Failed to generate PDF", errCode)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"report.pdf\"")
	w.Write(pdfData)
}
