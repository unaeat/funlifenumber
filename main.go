package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/web.html")
	})

	http.HandleFunc("/generate-pdf", handleGeneratePdf)
	http.HandleFunc("/generate-num", handleGenerateNum)
	http.HandleFunc("/generate-name", handleGenerateName)
	http.ListenAndServe(":8080", nil)
}

func handleGeneratePdf(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.FormValue("passport_name")
	nickName := r.FormValue("nickname")
	birthday := r.FormValue("birthday_time")

	pdfData, isLeap, errCode := generatePdf(name, nickName, birthday)
	if errCode != http.StatusOK {
		http.Error(w, "Failed to generate PDF", errCode)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("X-IsLeap", fmt.Sprint(isLeap))
	w.Write(pdfData)
}

func handleGenerateNum(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	num := r.FormValue("life_number")
	data, errCode := generateNum(num)
	if errCode != http.StatusOK {
		http.Error(w, "Failed to generate number", errCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleGenerateName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	num := r.FormValue("life_name")
	data, errCode := generateName(num)
	if errCode != http.StatusOK {
		http.Error(w, "Failed to generate number", errCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
