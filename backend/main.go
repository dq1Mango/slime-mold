// simple file server to serve all the files (wow)

package main

import (
	// "errors"
	"flag"
	"log"
	"net/http"
	// "os"
)

// func getRoot(w http.ResponseWriter, r *http.Request) {
// 	fmt.Printf("got / request\n")
// 	io.WriteString(w, "This is my website!\n")
// }
// func getHello(w http.ResponseWriter, r *http.Request) {
// 	fmt.Printf("got /hello request\n")
// 	io.WriteString(w, "Hello, HTTP!\n")
// }
//
// // func app(w http.ResponseWriter, r *http.Request) {
// // 	w.
// // }
//
// func main() {
// 	// fileServer := http.FileServer(http.Dir("./static"))
// 	fileServer := http.FileServer(http.Dir("./static"))
// 	http.Handle("/static/", http.StripPrefix("/static", fileServer))
//
// 	// http.HandleFunc("/", getRoot)
// 	http.HandleFunc("/hello", getHello)
//
// 	err := http.ListenAndServe(":3333", nil)
// 	if err != nil {
// 		panic(err)
// 	}
// }

var (
	listen = flag.String("listen", ":8080", "listen address")
	dir    = flag.String("dir", "../frontend/", "directory to serve")
)

func main() {
	flag.Parse()
	log.Printf("listening on %q...", *listen)
	err := http.ListenAndServe(*listen, http.FileServer(http.Dir(*dir)))
	log.Fatalln(err)
}
