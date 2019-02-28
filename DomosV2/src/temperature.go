package main

//---------------------------------------------------------------------------------------------------------------------------------------------------------
// Package   : temperature.go
// Port rest : 1000
// Gets      :
//              /eau
//              /air
//              /air_local
// Reponse   : {"type":"temperature","id":"air_local","id_sonde":"28-0417714350ff","value":"13.75","elapse":"918.770618ms","compteur":1,"version":"1.0.0"}
// BLOW vertigo
// Glenn Miller
//---------------------------------------------------------------------------------------------------------------------------------------------------------

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	//"path/filepath"
	"time"

	"github.com/yryz/ds18b20"
	"gopkg.in/yaml.v2"
)

// Structure du Json
type apiTemperature struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	IDSonde  string `json:"idsonde"`
	Value    string `json:"value"`
	FileMqtt string `json:"filemqtt"`
	Elapse   string `json:"elapse"`
	Compteur int    `json:"compteur"`
	Version  string `json:"version"`
}

// Structure pour le fichier de configuration
type LectureInit struct {
	Sondes struct {
		Air          string `yaml:"air"`
		Mqttair      string `yaml:"mqttair"`
		Airlocal     string `yaml:"airlocal"`
		Mqttairlocal string `yaml:"mqttairlocal"`
		Eau          string `yaml:"eau"`
		Mqtteau      string `yaml:"mqtteau"`
		Port         string `yaml:"port"`
	} `yaml:"Sondes"`
}

// Déclare la liste
var liste = map[string]*struct{ idds18, labelmqtt string }{
	"air":       {"", ""},
	"air_local": {"", ""},
	"eau":       {"", ""},
}

var (
	compteurEau       = 0
	compteurAir       = 0
	compteurAir_Local = 0
	version           = "1.0.0"
	lectureInit       LectureInit
)

// Fonction Init du programme
func init() {

	//filename := filepath.IsAbs("/root/DomosV2/src/Conf/config.yaml")
	yamlFile, err := ioutil.ReadFile("/root/DomosV2/src/Conf/config.yaml")

	if err != nil {
		fmt.Println("Erreur de lecture du fichier config.yaml")
		fmt.Println(err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(yamlFile, &lectureInit)

	if err != nil {
		fmt.Println("Erreur dans le parse du fichier config.yaml")
		fmt.Println(err)
		os.Exit(1)
	}

	liste["eau"].idds18 = lectureInit.Sondes.Eau
	liste["eau"].labelmqtt = lectureInit.Sondes.Mqtteau
	liste["air"].idds18 = lectureInit.Sondes.Air
	liste["air"].labelmqtt = lectureInit.Sondes.Mqttair
	liste["air_local"].idds18 = lectureInit.Sondes.Airlocal
	liste["air_local"].labelmqtt = lectureInit.Sondes.Mqttairlocal

}

// Fonction main
func main() {

	http.HandleFunc("/eau", func(w http.ResponseWriter, r *http.Request) {
		api(w, r, liste["eau"].idds18, liste["eau"].labelmqtt, "eau")
	})
	http.HandleFunc("/air", func(w http.ResponseWriter, r *http.Request) {
		api(w, r, liste["air"].idds18, liste["air"].labelmqtt, "air")
	})
	http.HandleFunc("/air_local", func(w http.ResponseWriter, r *http.Request) {
		api(w, r, liste["air_local"].idds18, liste["air_local"].labelmqtt, "air_local")
	})

	fmt.Println("Server Temperature listen port : " + lectureInit.Sondes.Port)
	http.ListenAndServe(":"+lectureInit.Sondes.Port, nil)
}

// Fonction de réponse HTTP
func api(w http.ResponseWriter, r *http.Request, id string, id_mqtt string, idstring string) {

	compteur := 0

	switch idstring {
	case "eau":
		compteurEau = compteurEau + 1
		compteur = compteurEau

	case "air":
		compteurAir = compteurAir + 1
		compteur = compteurAir

	case "air_local":
		compteurAir_Local = compteurAir_Local + 1
		compteur = compteurAir_Local

	default:
		fmt.Println("Je ne comprends pas le choix !!!")
	}

	// Démarre chrono
	start := time.Now().UTC()

	// Demande la température
	temperature, err := ds18b20.Temperature(id)
	temperatureString := fmt.Sprintf("%.2f", temperature)

	// Calcul le temps pour lecture température
	elapse := time.Since(start)
	elapseSeconds := fmt.Sprintf("%.3f", elapse.Seconds())

	// Test si nous avons eu un problème de lecture
	if err != nil {
		fmt.Println("Probleme lecture sonde " + idstring + " : " + id)
		fmt.Println(err)
	}

	// Création du JSON
	donnees := apiTemperature{
		Type:     "temperature",
		ID:       idstring,
		IDSonde:  id,
		Value:    temperatureString,
		FileMqtt: id_mqtt,
		Elapse:   elapseSeconds,
		Compteur: compteur,
		Version:  version,
	}

	jsonData, _ := json.Marshal(donnees)

	w.Header().Set("content-type", "application/json")
	w.Write([]byte(jsonData))
}
