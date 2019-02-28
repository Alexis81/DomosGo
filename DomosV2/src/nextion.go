package main

//https://www.itead.cc/wiki/Nextion_Instruction_Set#Format_of_Device_Return_Data

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/tarm/serial"
)

// apiTemperature : Structure du Json
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

// apiStatus : Structure du Json
type apiStatus struct {
	Pompe            int    `json:"pompe"`
	Electrolyse      int    `json:"electrolyse"`
	Lampe            int    `json:"lampe"`
	LampePortail     int    `json:"lampeportail"`
	TemperatureEau   string `json:"temperatureeau"`
	TemperatureAir   string `json:"temperatureair"`
	TemperatureLocal string `json:"temperaturelocal"`
	DateTime         string `json:"datetime"`
	DureeFiltration  int64  `json:"dureefiltration"`
}

func main() {

	// Configuration de l'interface serial
	config := &serial.Config{Name: "/dev/ttyS0", Baud: 9600}

	// Ouverture du port RS232
	s, err := serial.OpenPort(config)
	if err != nil {
		// stops execution
		log.Fatal(err)
	}

	// Fin de ligne de commande
	endCom := "\xFF\xFF\xFF"

	// Permet de baisser la lumière de l'afficheur
	_, err = s.Write([]byte("dim=100" + endCom))

	// Temps de mise en veille de l'écran
	_, err = s.Write([]byte("sleep=0" + endCom))

	// Temps de mise en veille de l'écran
	_, err = s.Write([]byte("thsp=" + endCom))

	// Permet de reveiller l'écran si pression
	_, err = s.Write([]byte("thup=1" + endCom))

	// Lance les goroutines
	go lecture(s)
	go ecriture(s)

	// Permet de faire tourner le programme en boucle
	for {

	}

}

// Permet la lecture du port serial
func lecture(s *serial.Port) {

	// golang reader interface
	r := bufio.NewReader(s)

	for {
		// reads until delimiter is reached
		data, err := r.ReadBytes('\xFF')

		// Supprime les 0XFF
		data = bytes.Trim(data, "\xFF")

		// Test si nous avons une erreur
		if err != nil {
			// stops execution
			log.Fatal(err)
		}

		// Si un bouton est appuyé sur l'écran
		switch string(data) {
		case "pbt0:ON":
			fmt.Println("Bouton 0 = ON")
			resp, _ := http.Get("http://192.168.1.22:8080/pompe/1")
			defer resp.Body.Close()

		case "pbt0:OFF":
			fmt.Println("Bouton 0 = OFF")
			resp, _ := http.Get("http://192.168.1.22:8080/pompe/0")
			defer resp.Body.Close()

		case "pbt1:ON":
			fmt.Println("Bouton 1 = ON")
			resp, _ := http.Get("http://192.168.1.22:8080/electro/1")
			defer resp.Body.Close()

		case "pbt1:OFF":
			fmt.Println("Bouton 1 = OFF")
			resp, _ := http.Get("http://192.168.1.22:8080/electro/0")
			defer resp.Body.Close()

		case "pbt2:ON":
			fmt.Println("Bouton 2 = ON")
			resp, _ := http.Get("http://192.168.1.22:8080/lampe/1")
			defer resp.Body.Close()

		case "pbt2:OFF":
			fmt.Println("Bouton 2 = OFF")
			resp, _ := http.Get("http://192.168.1.22:8080/lampe/0")
			defer resp.Body.Close()

		case "pbt3:ON":
			fmt.Println("Bouton 3 = ON")
			resp, _ := http.Get("http://192.168.1.22:8080/lampe/1")
			defer resp.Body.Close()

		case "pbt3:OFF":
			fmt.Println("Bouton 3 = OFF")
			resp, _ := http.Get("http://192.168.1.22:8080/lampe/0")
			defer resp.Body.Close()
		}
	}
}

// Permet d'envoyer des informations à l'afficheur
func ecriture(s *serial.Port) {

	// Fin de ligne de commande
	endCom := "\xFF\xFF\xFF"

	// Déclare un flag
	flag := false

	for {

		// Température Air
		rs, _ := http.Get("http://192.168.1.22:8080/status")

		defer rs.Body.Close()

		if rs.StatusCode == 200 {
			//ms.Pompe, msg.Electrolyse, msg.Lampe, msg.LampePortail, msg.TemperatureEau, msg.TemperatureAir, msg.TemperatureLocal, msg.DateTime, msg.DureeFiltration
			pompe, electrolyse, lampe, lampePortail, temperatureEau, temperatureAir, temperatureLocal, dateTime, _ := httpStatus(rs)

			_, err := s.Write([]byte("Filtration.txt=\"" + dateTime + "\"" + endCom))

			_, err = s.Write([]byte("TempEau.txt=\"" + temperatureEau + "\"" + endCom))
			_, err = s.Write([]byte("TempAir.txt=\"" + temperatureAir + "\"" + endCom))
			_, err = s.Write([]byte("TempLocal.txt=\"" + temperatureLocal + "\"" + endCom))

			if err != nil {
				fmt.Println("error send lampe portail")
			}

			if pompe == "0" {
				_, err := s.Write([]byte("bt0.val=0" + endCom))

				if err != nil {
					fmt.Println("error send pompe")
				}
			} else {
				_, err := s.Write([]byte("bt0.val=1" + endCom))

				if err != nil {
					fmt.Println("error send pompe")
				}
			}

			if electrolyse == "0" {
				_, err := s.Write([]byte("bt1.val=0" + endCom))

				if err != nil {
					fmt.Println("error send electrolyse")
				}
			} else {
				_, err := s.Write([]byte("bt1.val=1" + endCom))

				if err != nil {
					fmt.Println("error send electrolyse")
				}
			}

			if lampe == "0" {
				_, err := s.Write([]byte("bt2.val=0" + endCom))
				if err != nil {
					fmt.Println("error send lampe")
				}
			} else {
				_, err := s.Write([]byte("bt2.val=1" + endCom))
				if err != nil {
					fmt.Println("error send lampe")
				}
			}

			if lampePortail == "0" {
				_, err := s.Write([]byte("bt3.val=0" + endCom))

				if err != nil {
					fmt.Println("error send lampe portail")
				}
			} else {
				_, err := s.Write([]byte("bt3.val=1" + endCom))

				if err != nil {
					fmt.Println("error send lampe portail")
				}
			}

		}

		if flag == false {
			_, err := s.Write([]byte("p1.pic=4" + endCom))

			if err != nil {
				fmt.Println("error send wifi")
			}
			flag = true
		} else {
			_, err := s.Write([]byte("p1.pic=3" + endCom))

			if err != nil {
				fmt.Println("error send wifi")
			}
			flag = false
		}

		duration := time.Duration(1) * time.Second // Pause for 10 seconds
		time.Sleep(duration)
	}
}

// Permet d'interroger l'API température *************************************
func httpTemperature(rs *http.Response) (string, string, string, string) {

	bodyBytes, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		fmt.Println("error send")
	}

	// Unmarshal
	var msg apiTemperature
	err = json.Unmarshal(bodyBytes, &msg)
	if err != nil {
		fmt.Println("error send Temperatures")
	}

	return msg.FileMqtt, msg.Value, msg.Elapse, strconv.Itoa(msg.Compteur)
}

// Permet d'interroger l'API status *************************************
func httpStatus(rs *http.Response) (string, string, string, string, string, string, string, string, string) {

	bodyBytes, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		fmt.Println("error send Status")
	}

	// Unmarshal
	var msg apiStatus
	err = json.Unmarshal(bodyBytes, &msg)
	if err != nil {
		fmt.Println("error Unmarshal status")
	}

	return strconv.Itoa(msg.Pompe), strconv.Itoa(msg.Electrolyse), strconv.Itoa(msg.Lampe), strconv.Itoa(msg.LampePortail), msg.TemperatureEau, msg.TemperatureAir, msg.TemperatureLocal, msg.DateTime, strconv.FormatInt(msg.DureeFiltration, 10)
}
