package main

//---------------------------------------------------------------------------------------------------------------------------------------------------------
// Package   : statsRaspberry.go
// Port rest : 1003
// Gets      :
//              /StatsNetatmo
//
// netatmo.intérieur.co2 : 843
// netatmo.intérieur.humidity : 52
// netatmo.intérieur.noise : 43
// netatmo.intérieur.pressure : 995.6
// netatmo.intérieur.temperature : 22.8
//
//---------------------------------------------------------------------------------------------------------------------------------------------------------

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	netatmo "github.com/romainbureau/netatmo-api-go"
	"gopkg.in/yaml.v2"
)

var (
	lectureInit LectureInit
	version     = "1.0.0"
)

// Structure du Json
type StatsNetatmo struct {
	Type                                       string `json:"type"`
	Domos_poolhouse_bureau_netatmo_Co2         string `json:"Domos.poolhouse.bureau.netatmo.Co2"`
	Domos_poolhouse_bureau_netatmo_Humidity    string `json:"Domos.poolhouse.bureau.netatmo.Humidity"`
	Domos_poolhouse_bureau_netatmo_Noise       string `json:"Domos.poolhouse.bureau.netatmo.Noise"`
	Domos_poolhouse_bureau_netatmo_Pressure    string `json:"Domos.poolhouse.bureau.netatmo.Pressure"`
	Domos_poolhouse_bureau_netatmo_Temperature string `json:"Domos.poolhouse.bureau.netatmo.Temperature"`
	Domos_poolhouse_bureau_netatmo_Elapsed     string `json:"Domos.poolhouse.bureau.netatmo.Elapsed"`
	Version                                    string `json:"version"`
}

// Structure pour le fichier de configuration
type LectureInit struct {
	Netatmo struct {
		Clientid     string `yaml:"clientid"`
		Clientsecret string `yaml:"clientsecret"`
		Username     string `yaml:"username"`
		Password     string `yaml:"password"`
		Url          string `yaml:"url"`
		Port         string `yaml:"port"`
	} `yaml:"Netatmo"`
}

//--------------------------------------------------------------------------------------------------------------------------------
// Fonction : main
//
//--------------------------------------------------------------------------------------------------------------------------------
func main() {

	// Lecture du fichier config.yaml
	//filename := filepath.IsAbs("/root/DomosV2/src/Conf/config.yaml")
	yamlFile, err := ioutil.ReadFile("/root/DomosV2/src/Conf/config.yaml")
	if err != nil {
		fmt.Println("Erreur de lecture du fichier config.yaml")
		fmt.Println(err)
		os.Exit(1)
	}

	// Parse du fichier config.yaml
	err = yaml.Unmarshal(yamlFile, &lectureInit)
	if err != nil {
		fmt.Println("Erreur dans le parse du fichier config.yaml")
		fmt.Println(err)
		os.Exit(1)
	}

	http.HandleFunc(lectureInit.Netatmo.Url, func(w http.ResponseWriter, r *http.Request) {
		api(w, r, lectureInit.Netatmo.Clientid, lectureInit.Netatmo.Clientsecret, lectureInit.Netatmo.Username, lectureInit.Netatmo.Password)
	})

	fmt.Println("Server StatsNetatmo listen port : " + lectureInit.Netatmo.Port)
	http.ListenAndServe(":"+lectureInit.Netatmo.Port, nil)
}

//--------------------------------------------------------------------------------------------------------------------------------
// Fonction : api
// Entrées  : w, r, clientid, clientsecret, username, password
// Sortie   :
//--------------------------------------------------------------------------------------------------------------------------------
func api(w http.ResponseWriter, r *http.Request, clientid string, clientsecret string, username string, password string) {

	// Démarre chrono
	start := time.Now().UTC()

	// Créer un client Netatmo
	netatmoClient, err := netatmo.NewClient(netatmo.Config{
		ClientID:     clientid,
		ClientSecret: clientsecret,
		Username:     username,
		Password:     password,
	})

	if err != nil {
		log.Println(err.Error())
	}

	// Lecture des valeurs
	values, err := netatmoClient.Read()
	if err != nil {
		log.Println(err.Error())
	}

	// Extraction des valeurs
	var Values = make(map[string]float32)
	for _, station := range values.Stations() {
		for _, module := range station.Modules() {
			_, data := module.Data()
			moduleName := strings.ToLower(module.ModuleName)
			for key, value := range data {
				metricName := fmt.Sprintf("netatmo.%s.%s", moduleName, key)
				//fmt.Print(metricName)
				//fmt.Print(" --> ")
				Values[metricName] = value.(float32)
				//fmt.Println(value.(float32))
			}
		}
	}

	// Calcul le temps
	elapse := time.Since(start)
	elapseSeconds := fmt.Sprintf("%.3f", elapse.Seconds())

	// Création du JSON
	donnees := StatsNetatmo{
		Type: "StatNetatmo",
		Domos_poolhouse_bureau_netatmo_Co2:         fmt.Sprintf("%v", Values["netatmo.intérieur.co2"]),
		Domos_poolhouse_bureau_netatmo_Humidity:    fmt.Sprintf("%v", Values["netatmo.intérieur.humidity"]),
		Domos_poolhouse_bureau_netatmo_Noise:       fmt.Sprintf("%v", Values["netatmo.intérieur.noise"]),
		Domos_poolhouse_bureau_netatmo_Pressure:    fmt.Sprintf("%v", Values["netatmo.intérieur.pressure"]),
		Domos_poolhouse_bureau_netatmo_Temperature: fmt.Sprintf("%v", Values["netatmo.intérieur.temperature"]),
		Domos_poolhouse_bureau_netatmo_Elapsed:     elapseSeconds,
		Version: version,
	}

	// Création du JSON
	jsonData, _ := json.Marshal(donnees)

	w.Header().Set("content-type", "application/json")
	w.Write([]byte(jsonData))

}
