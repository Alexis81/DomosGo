package main

//---------------------------------------------------------------------------
//
//
//---------------------------------------------------------------------------
// GPIO :
//        17 -> lampe piscine
//        18 -> Electrolyse
//        22 -> Lampes PoolHouse
//        23 -> Sondes DS18B
//        24 -> Led rouge
//        25 -> Pompe piscine
//
// Clé API Live Objetcs : d3c3aa0f77fe4e388ded2c529d1746f3
// https://liveobjects.orange-business.com/#/datastore?pageSize=20&pageNumber=1
//
//---------------------------------------------------------------------------
// Versions :
//            1.3 : Ajout des métriques pour netdata (01/04/2018)
//            1.4 : Ajout température extérieure (02/04/2018)
//            1.5 : Punlication sur Live Objects (29/06/2018)
//
//---------------------------------------------------------------------------

import (
	"bufio"
	"encoding/json"
	_ "expvar"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/hhkbp2/go-strftime"
	"github.com/stianeikeland/go-rpio"
	"github.com/tarm/serial"
	"github.com/vjeantet/jodaTime" // https://github.com/vjeantet/jodaTime
	"github.com/zpatrick/go-config"
)

var (
	//err                          error
	c                            mqtt.Client
	cheminWriteFichier           = ""
	cmdPortail                   = ""
	compteurAir                  = ""
	compteurAirLocal             = ""
	compteurEau                  = ""
	data                         string
	dataAir                      = ""
	dataAirLocal                 = ""
	dataEau                      = ""
	datamqtt                     map[string]interface{}
	datamqttPortail              map[string]interface{}
	dateTime                     string
	DsAir                        = ""
	DsAirLocal                   = ""
	DsEau                        = ""
	duree                        float64
	dureeFiltration              float64
	dureeMetrcisInt              time.Duration
	dureeMetrcisString           = ""
	dureeWriteFichier            = ""
	dureeWriteFichierInt         time.Duration
	elapseAir                    = ""
	elapseAirLocal               = ""
	elapseEau                    = ""
	etatElectrolyse              = 0
	etatLampe                    = 0
	etatLampePortail             = 0
	etatPompe                    = 0
	etatPortail                  = ""
	exporterName                 string
	flagPlage1                   = true
	flagPlage2                   = true
	flagPlage3                   = true
	formattedPlage1              string
	formattedPlage1Fin           string
	formattedPlage2              string
	formattedPlage2Fin           string
	formattedPlage3              string
	formattedPlage3Fin           string
	FrequenceLedString           = ""
	FrequenceLedTimeDuration     time.Duration
	FrequenceMesuresString       = ""
	FrequenceMesuresTimeDuration time.Duration
	grafanaActif                 = true
	graphiteHost                 = "92.154.38.115"
	graphitePort                 = 2003
	ifttKey                      = ""
	input                        = ""
	lampePortail                 = ""
	mapDate                      = make(map[string]string)
	MesureTemperaturesUint       uint
	pinElectrolyse               = rpio.Pin(18)
	pinLampe                     = rpio.Pin(22)
	pinLampePiscine              = rpio.Pin(17)
	pinLed                       = rpio.Pin(24)
	pinPompe                     = rpio.Pin(25)
	Plage1                       = ""
	Plage2                       = ""
	Plage3                       = ""
	portMqtt                     = ""
	PortWeb                      = ""
	salonHumidite                = ""
	salonTemperature             = ""
	serialDevice                 string
	serveurMqtt                  = ""
	tAirFloat                    float64
	tAirLocalFloat               float64
	tEauFloat                    float64
	telegramId                   = 0
	telegramToken                = ""
	teleinfoActif                = true
	TeleinfoLectureTimeDuration  time.Duration
	teleinfoString               = ""
	temperatureAirLocalString    = ""
	temperatureAirString         = ""
	temperatureEauString         = ""
	TemperatureElectrolyseInt    int
	texte                        = ""
	TPlage1                      time.Time
	TPlage2                      time.Time
	TPlage3                      time.Time
	value                        string
	netatmoActif                 bool
	cleApiLiveObjects            string
	urlLiveObjects               string
	topicPushLiveObjects         string
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

// MyStruct : Structure pour le compteur de durée de filtration
type MyStruct struct {
	Flag     bool
	compteur int64
	t        time.Duration
}

var ms = MyStruct{}

// Message : Structure pour connaitre état des différents paramètres
type Message struct {
	Pompe            int
	Electrolyse      int
	Lampe            int
	LampePortail     int
	TemperatureEau   string
	TemperatureAir   string
	TemperatureLocal string
	DateTime         string
	DureeFiltration  time.Duration
}

// Fonction pour initialiser le programme
func init() {

	nprocess := runtime.GOMAXPROCS(runtime.NumCPU())
	dateTime = jodaTime.Format("dd/MM/YYY HH:mm:ss", time.Now())

	mapDate["Pompe"] = ""
	mapDate["Electrolyse"] = ""
	mapDate["lampe"] = ""
	mapDate["LampePiscine"] = ""
	mapDate["LampePortail"] = ""

	fmt.Println("---------------------------------------------------------")
	fmt.Println("                Golang to DOMOS")
	fmt.Println("---------------------------------------------------------")
	fmt.Print("Lancement : ")
	fmt.Println(dateTime)

	fmt.Println(" * Start micro-service : temperature")

	goTemperature := exec.Command("nohup", "go", "run", "/root/DomosV2/src/temperature.go")
	err := goTemperature.Start()
	gestionErr("Problème lancement du micro service températures", err)

	fmt.Println(" * Start micro-service : statsRaspberry")
	goStats := exec.Command("nohup", "go", "run", "/root/DomosV2/src/statsRaspberry.go")
	err = goStats.Start()
	gestionErr("Problème lancement du micro service statistiques Raspberry", err)

	fmt.Println(" * Start micro-service : statsNetatmo")
	goStatsNetatmo := exec.Command("nohup", "go", "run", "/root/DomosV2/src/netatmo.go")
	err = goStatsNetatmo.Start()
	gestionErr("Problème lancement du micro service statistiques Netatmo", err)

	time.Sleep(5000 * time.Millisecond)

	// Lecture du fichier Conf/config.ini
	fmt.Println(" - Lecture du fichier config.ini")
	lectureIni()

	ms.compteur, ms.t = LireFichier(cheminWriteFichier)

	fmt.Println(ms.t)

	if teleinfoActif {
		// Paramètres pour la communication série pour Téléinfo EDF
		flag.StringVar(&serialDevice, "device", "/dev/ttyS0", "Serial port to read frames from")
		flag.Parse()

		port, _ := OpenPort(serialDevice)

		// Lecture du fichier Conf/config.ini
		fmt.Println(" - Lancement routine Téléinfo EDF")
		go lectureFrame(port)

	} else {
		fmt.Println("!!! Pas de lecture compteur EDF")
	}

	// Métriques de GO
	go metriquesGolang()

	// Métriques du Raspberry PI 3
	go statCPU()

	// Métriques de Netatmo
	if netatmoActif {
		go Netatmo()
	}

	// Permet d'écrire le fichier de duration filtration
	go writeFichierDurationFiltration()

	// Se connecter au serveur MQTT
	fmt.Println(" - Abonnements au serveur MQTT")
	mqttSubscribe()

	// Etre certains que la lampe portail soit sur OFF
	etatLampePortail = 0
	//http.Get(lampePortail + "inter1_off")
	publishMqtt("/domos/portail/light/cmd/", "OFF1")

	// Nombres de processeurs
	fmt.Print(" - Nombres de processeurs : ")
	fmt.Println(nprocess)

}

// main : Fonction principal du programme
//---------------------------------------------------------------------------
func main() {
	// Expose métrique de Golang pour netdata
	go http.ListenAndServe(":8085", nil)

	// Démarre le compteur pour la durée de filtration de la piscine
	go ms.counter()

	// Lancement lecture clavier en goroutine
	fmt.Println(" - Scan du clavier")
	go clavier()

	fmt.Println(" - Affectation des GPIO")
	if err := rpio.Open(); err != nil {
		fmt.Println("Impossible d'affecter les GPIO")
		fmt.Println(err)
		os.Exit(1)
	}

	// Unmap gpio memory when done
	defer rpio.Close()

	// Affecte les pins en sortie
	pinLed.Output()
	pinPompe.Output()
	pinElectrolyse.Output()
	pinLampe.Output()

	// Passe les pins à Off
	pinPompe.Low()
	pinElectrolyse.Low()
	pinLampe.Low()

	fmt.Println(" - Lancement des Jobs")

	// Lance la lecture des sondes
	go LectureSondes()

	// Permet de faire clignter la led de vie
	go BlinkLed()

	// Create a mux for routing incoming requests
	m := http.NewServeMux()

	m.HandleFunc("/html/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.URL.Path[1:])
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	// All URLs will be handled by this function
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<head>"+
			"<meta name='viewport' content='width=device-width, initial-scale=1'>"+
			"<meta charset='UTF-8'>"+
			"<link rel='stylesheet' href='https://maxcdn.bootstrapcdn.com/bootstrap/3.3.5/css/bootstrap.min.css'>"+
			"<link rel='stylesheet' href='//cdnjs.cloudflare.com/ajax/libs/bootstrap-table/1.8.1/bootstrap-table.min.css'>"+

			"<h1><strong>Domos by Golang</strong></h1>"+

			"<table class='table table-striped'>"+
			"<thead>"+
			"<tr>"+
			"<th></th>"+
			"<th>Commandes</th>"+
			"<th>Aide</th>"+
			"</tr>"+
			"</thead>"+
			"<tbody>"+
			"<tr>"+
			"<th scope='row'>1</th>"+
			"<td>Températures</td>"+
			"<td>/temps</td>"+
			"</tr>"+

			"<th scope='row'>2</th>"+
			"<td>Pompe piscine On</td>"+
			"<td>/pompe/1 </td>"+
			"</tr>"+

			"<th scope='row'>3</th>"+
			"<td>Pompe piscine Off</td>"+
			"<td>/pompe/0</td>"+
			"</tr>"+

			"<th scope='row'>4</th>"+
			"<td>Electrolyse On</td>"+
			"<td>/electro/1</td>"+
			"</tr>"+

			"<th scope='row'>5</th>"+
			"<td>Electrolyse Off</td>"+
			"<td>/electro/0</td>"+
			"</tr>"+

			"<th scope='row'>6</th>"+
			"<td>Lampes extérieures On</td>"+
			"<td>/lampe/1</td>"+
			"</tr>"+

			"<th scope='row'>7</th>"+
			"<td>Lampes extérieures On</td>"+
			"<td>/lampe/0</td>"+
			"</tr>"+

			"<th scope='row'>8</th>"+
			"<td>Lampes portail On</td>"+
			"<td>/lampePortail/1</td>"+
			"</tr>"+

			"<th scope='row'>9</th>"+
			"<td>Lampes portail Off</td>"+
			"<td>/lampePortail/0</td>"+
			"</tr>"+

			"<th scope='row'>10</th>"+
			"<td>Prise Off</td>"+
			"<td>/prise/0</td>"+
			"</tr>"+

			"<th scope='row'>11</th>"+
			"<td>Prise On</td>"+
			"<td>/prise/1</td>"+
			"</tr>"+

			"</tbody>"+
			"</table>")
	})

	m.HandleFunc("/etat", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<head>"+
			"<meta name='viewport' content='width=device-width, initial-scale=1'>"+
			"<meta charset='UTF-8'>"+
			"<link rel='stylesheet' href='https://maxcdn.bootstrapcdn.com/bootstrap/3.3.5/css/bootstrap.min.css'>"+
			"</head>"+

			"<div class='container'>"+
			"<div class='row'>"+
			"<div class='col-md-12'>"+
			"<h3 class='text-left'>"+
			"Golang Domos"+
			"</h3>"+
			"</div>"+
			"</div>"+
			"<div class='row'>"+
			"<div class='col-md-6'>"+
			"<ul class='nav nav-pills'>"+
			"<li class='active'>"+
			"<a href='#'> <span class='badge pull-right'>%.2f°C</span> Eau</a>"+
			"</li>"+
			"</ul>"+
			"<ul class='nav nav-pills'>"+
			"<li class='active'>"+
			"<a href='#'> <span class='badge pull-right'>%.2f°C</span> Air</a>"+
			"</li>"+
			"<li>"+
			"<a href='#'></a>"+
			"</li>"+
			"</ul>"+
			"</div>"+
			"<div class='col-md-6'>"+

			"<button type='button' class='btn btn-success'>"+
			"Pompe"+
			"</button>"+
			"<button type='button' class='btn btn-danger'>"+
			"Electrolyse"+
			"</button>"+
			"</div>"+
			"</div>"+
			"</div>", tEauFloat, tAirFloat)
	})

	m.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		dateTime = jodaTime.Format("dd/MM/YYY HH:mm:ss", time.Now())
		m := Message{etatPompe, etatElectrolyse, etatLampe, etatLampePortail, temperatureEauString, temperatureAirString, temperatureAirLocalString, dateTime, ms.t}
		b, _ := json.Marshal(m)
		fmt.Fprintf(w, string(b))
	})

	m.HandleFunc("/temps", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Temp&eacute;rature eau       : %.2f°C</li>"+
			"<li>Temp&eacute;rature air       : %.2f°C</li>"+
			"<li>Temp&eacute;rature air local : %.2f°C</li>"+
			"<li>Etat pompe eau               : %d - (%s)</li>"+
			"<li>Etat electrolyse             : %d - (%s)</li>"+
			"<li>Etat Lampe                   : %d - (%s)</li>"+
			"<li></li>"+
			"<li>Filtration -----------------------</li>"+
			"<li>Durée                  : %.f heures</li>"+
			"<li>Plage 1 début          : %s</li>"+
			"<li>Plage 1 fin            : %s</li>"+
			"<li>Plage 2 début          : %s</li>"+
			"<li>Plage 2 fin            : %s</li>"+
			"<li>Plage 3 début          : %s</li>"+
			"<li>Plage 3 fin            : %s</li>"+
			"<li>----------------------------------</li>"+
			"<li>Durée de fonctionnement : %s</li>"+
			"</ul>", tEauFloat, tAirFloat, tAirLocalFloat, etatPompe, mapDate["Pompe"], etatElectrolyse, mapDate["Electrolyse"], etatLampe, mapDate["Lampe"], duree, formattedPlage1, formattedPlage1Fin, formattedPlage2, formattedPlage2Fin, formattedPlage3, formattedPlage3Fin, ms.t)
	})

	m.HandleFunc("/pompe/1", func(w http.ResponseWriter, r *http.Request) {
		etatPompe = 1
		pinPompe.High()
		fmt.Println("Pompe : On")
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat pompe eau         : %d</li>"+
			"</ul>", etatPompe)
	})

	m.HandleFunc("/pompe/0", func(w http.ResponseWriter, r *http.Request) {
		etatPompe = 0
		pinPompe.Low()
		etatElectrolyse = 0
		pinElectrolyse.Low()
		fmt.Println("Pompe : Off")
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat pompe eau         : %d</li>"+
			"</ul>", etatPompe)
	})

	m.HandleFunc("/electro/1", func(w http.ResponseWriter, r *http.Request) {
		if etatPompe == 1 {
			electrolyseOn()
			fmt.Println("Electrolyse : On")
			fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
				"<ul>"+
				"<li>Etat electrolyse         : %d</li>"+
				"</ul>", etatElectrolyse)
		} else {
			fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
				"<ul>"+
				"<li>Impossible de lancer electrolyse, pompe Off</li>"+
				"</ul>")
		}
	})

	m.HandleFunc("/electro/0", func(w http.ResponseWriter, r *http.Request) {
		etatElectrolyse = 0
		pinElectrolyse.Low()
		fmt.Println("Electrolyse : Off")
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat electrolyse         : %d</li>"+
			"</ul>", etatElectrolyse)
	})

	m.HandleFunc("/lampe/0", func(w http.ResponseWriter, r *http.Request) {
		etatLampe = 0
		pinLampe.Low()
		fmt.Println("Lampe PoolHouse sur : OFF")
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat lampes         : %d</li>"+
			"</ul>", etatLampe)
	})

	m.HandleFunc("/lampe/1", func(w http.ResponseWriter, r *http.Request) {
		etatLampe = 1
		pinLampe.High()
		fmt.Println("Lampe PoolHouse sur : ON")
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat lampes         : %d</li>"+
			"</ul>", etatLampe)
	})

	m.HandleFunc("/lampePortail/0", func(w http.ResponseWriter, r *http.Request) {
		etatLampePortail = 0
		fmt.Println("Portail : Off")
		//http.Get(lampePortail + "inter1_off")
		pushbulletSend("Domos", "Lampe portail Off")
		publishMqtt("cmnd/sonoff/power", "off")
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat lampes portail        : %d</li>"+
			"</ul>", etatLampePortail)
	})

	m.HandleFunc("/lampePortail/1", func(w http.ResponseWriter, r *http.Request) {
		etatLampePortail = 1
		fmt.Println("Lampe portail : On")
		//http.Get(lampePortail + "inter1_on")
		pushbulletSend("Domos", "Lampe portail On")
		publishMqtt("cmnd/sonoff/power", "on")
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat lampes portail        : %d</li>"+
			"</ul>", etatLampePortail)
	})

	m.HandleFunc("/Portail/0", func(w http.ResponseWriter, r *http.Request) {
		etatLampePortail = 0
		fmt.Println("Portail : Off")
		publishMqtt("/domos/portail/cmd", "OFF")
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat portail        : %d</li>"+
			"</ul>", etatLampePortail)
	})

	m.HandleFunc("/Portail/1", func(w http.ResponseWriter, r *http.Request) {
		etatLampePortail = 1
		fmt.Println("portail : On")
		publishMqtt("/domos/portail/cmd", "ON")
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat portail        : %d</li>"+
			"</ul>", etatLampePortail)
	})

	m.HandleFunc("/prise/0", func(w http.ResponseWriter, r *http.Request) {
		iftt("EteindrePrise")
		fmt.Println("Prise sur : OFF")
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Prise         : Off</li>"+
			"</ul>")
	})

	m.HandleFunc("/prise/1", func(w http.ResponseWriter, r *http.Request) {
		iftt("AllumePrise")
		fmt.Println("Prise sur : ON")
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Prise         : On</li>"+
			"</ul>")
	})

	// Create a server listening on port 8000
	t := &http.Server{
		Addr:    ":" + PortWeb,
		Handler: m,
	}

	ipadresse := Ipadresse()

	fmt.Println(" - Lancement du serveur Web : http://" + ipadresse + ":" + PortWeb)
	// Continue to process new requests until an error occurs
	go t.ListenAndServe()

	//Affiche durée totale de filtration
	fmt.Print(" - Durée totale de filtration : ")
	fmt.Println(ms.t)

	fmt.Println(" - Run du programme...")

	// Boucle sans fin ------------------------------------------------------
	for {
		heureCourante := strftime.Format("%H:%M", time.Now())

		// Calucl la durée des plages
		calculPlages()

		if heureCourante > formattedPlage1 && heureCourante < formattedPlage1Fin && flagPlage1 {
			fmt.Println("- Plage 1")
			pompeOn()
			electrolyseOn()
			pushbulletSend("Domos", "Run Plage 1 : "+temperatureEauString+" °c")
			flagPlage1 = false
			flagPlage2 = true
			flagPlage3 = true
		}

		if heureCourante > formattedPlage1Fin && !flagPlage1 {
			fmt.Println("- Plage 1 fin")
			pompeOff()
			electrolyseOff()
			pushbulletSend("Domos", "Off Plage 1 : "+temperatureEauString+" °c")
			flagPlage1 = true
		}

		if heureCourante > formattedPlage2 && heureCourante < formattedPlage2Fin && flagPlage2 {
			fmt.Println("- Plage 2")
			pompeOn()
			electrolyseOn()
			pushbulletSend("Domos", "Run Plage 2 : "+temperatureEauString+" °c")
			flagPlage1 = true
			flagPlage2 = false
			flagPlage3 = true
		}

		if heureCourante > formattedPlage2Fin && !flagPlage2 {
			fmt.Println("- Plage 2 fin")
			pompeOff()
			electrolyseOff()
			pushbulletSend("Domos", "Off Plage 2 : "+temperatureEauString+" °c")
			flagPlage2 = true
		}

		if heureCourante > formattedPlage3 && heureCourante < formattedPlage3Fin && flagPlage3 {
			fmt.Println("- Plage 3")
			pompeOn()
			electrolyseOn()
			pushbulletSend("Domos", "Run Plage 3 : "+temperatureEauString+" °c")
			flagPlage1 = true
			flagPlage2 = true
			flagPlage3 = false
		}

		if heureCourante > formattedPlage3Fin && !flagPlage3 {
			fmt.Println("- Plage 3 fin")
			pompeOff()
			electrolyseOff()
			pushbulletSend("Domos", "Off Plage 3 : "+temperatureEauString+" °c")
			flagPlage3 = true
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// LectureSondes : Lire les sondes
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func LectureSondes() {

	var liste = make(map[string]string)

	liste["Pompe"] = "Domos.poolhouse.localTechnique.Pompe"
	liste["Electrolyse"] = "Domos.poolhouse.localTechnique.Electrolyse"
	liste["Lampe"] = "Domos.poolhouse.localTechnique.Lampe"

	for {
		//--- API TEMPERATURES -----------------------------------------------------------------------------------------
		// Lecture des sondes
		rs, err := http.Get("http://192.168.1.22:1000/eau")
		// Process response
		gestionErr("Erreur de lecture de la sonde Eau", err)
		defer rs.Body.Close()

		if rs.StatusCode == 200 {
			dataEau, temperatureEauString, elapseEau, compteurEau = httpTemperature(rs)
			tEauFloat, _ = strconv.ParseFloat(temperatureEauString, 64)
		}
		rs, err = http.Get("http://192.168.1.22:1000/air")
		// Process response
		gestionErr("Erreur de lecture de la sonde Air", err)
		defer rs.Body.Close()

		if rs.StatusCode == 200 {
			dataAir, temperatureAirString, elapseAir, compteurAir = httpTemperature(rs)
			tAirFloat, _ = strconv.ParseFloat(temperatureAirString, 64)
		}
		rs, err = http.Get("http://192.168.1.22:1000/air_local")
		// Process response
		gestionErr("Erreur de lecture de la sonde Local Technique", err)
		defer rs.Body.Close()
		if rs.StatusCode == 200 {
			dataAirLocal, temperatureAirLocalString, elapseAirLocal, compteurAirLocal = httpTemperature(rs)
			tAirLocalFloat, _ = strconv.ParseFloat(temperatureAirLocalString, 64)
		}
		//--------------------------------------------------------------------------------------------------------------

		dureeFiltration = calculDuree()

		// Insertion des données dans graphite
		err = GraphiteCnx(dataEau, temperatureEauString, graphiteHost, graphitePort)
		errGraphique(err, "1")

		PushLiveObject(temperatureEauString)

		// Insertion des données dans graphite
		err = GraphiteCnx(dataEau+"Elapse", elapseEau, graphiteHost, graphitePort)
		errGraphique(err, "1bis")

		// Insertion des données dans graphite
		err = GraphiteCnx(dataEau+"Compteur", compteurEau, graphiteHost, graphitePort)
		errGraphique(err, "1bis")

		// Insertion des données dans graphite
		err = GraphiteCnx(dataAir, temperatureAirString, graphiteHost, graphitePort)
		errGraphique(err, "2")

		// Insertion des données dans graphite
		err = GraphiteCnx(dataAir+"Elapse", elapseAir, graphiteHost, graphitePort)
		errGraphique(err, "2bis")

		// Insertion des données dans graphite
		err = GraphiteCnx(dataAir+"Compteur", compteurAir, graphiteHost, graphitePort)
		errGraphique(err, "2bis")

		err = GraphiteCnx(dataAirLocal, temperatureAirLocalString, graphiteHost, graphitePort)
		errGraphique(err, "3")

		err = GraphiteCnx(dataAirLocal+"Elapse", elapseAirLocal, graphiteHost, graphitePort)
		errGraphique(err, "3bis")

		err = GraphiteCnx(dataAirLocal+"Compteur", compteurAirLocal, graphiteHost, graphitePort)
		errGraphique(err, "3bis")

		data = liste["Pompe"]
		value = strconv.Itoa(etatPompe)
		err = GraphiteCnx(data, value, graphiteHost, graphitePort)
		errGraphique(err, "4")

		data = liste["Electrolyse"]
		value = strconv.Itoa(etatElectrolyse)
		err = GraphiteCnx(data, value, graphiteHost, graphitePort)
		errGraphique(err, "5")

		data = liste["Lampe"]
		value = strconv.Itoa(etatLampe)
		err = GraphiteCnx(data, value, graphiteHost, graphitePort)
		errGraphique(err, "6")

		time.Sleep(FrequenceMesuresTimeDuration * time.Millisecond)
	}
}

// Permet d'interroger l'API température *************************************
func httpTemperature(rs *http.Response) (string, string, string, string) {

	bodyBytes, err := ioutil.ReadAll(rs.Body)
	gestionErr("Impossible d'interroger l'API Température", err)

	// Unmarshal
	var msg apiTemperature
	err = json.Unmarshal(bodyBytes, &msg)
	gestionErr("Impossible de décoder le Json API Température", err)

	return msg.FileMqtt, msg.Value, msg.Elapse, strconv.Itoa(msg.Compteur)
}

// Gestion des erreurs dans Graphite
func errGraphique(err error, numero string) {
	if err != nil {
		fmt.Println("Erreur insertion dans graphite" + numero)
	}
}

// BlinkLed : Faire clignoter la led
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func BlinkLed() {
	for {
		pinLed.Toggle()
		time.Sleep(FrequenceLedTimeDuration * time.Millisecond)
	}
}

// lectureIni : Permet de lire le fichier ini et placer les valeurs dans des variables
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func lectureIni() {
	// Lecture des valeurs dans le fichier de config qui se trouve ./config/config.ini
	iniFile := config.NewINIFile("Conf/config.ini")
	c := config.NewConfig([]config.Provider{iniFile})

	// Test si nous avons bien trouvé le fichier
	if err := c.Load(); err != nil {
		fmt.Println("Impossible de lire le fichier de conf : ")
		fmt.Println(err)

		os.Exit(1)
	}

	// Affectation des variables plage horaire
	FrequenceLedString, err := c.String("led.Frequence")
	gestionErr("Impossible de lire valeur led.Frequence dans le fichier ini", err)

	FrequenceMesuresString, err = c.String("mesures.EchantillonTemperature")
	gestionErr("Impossible de lire valeur mesures.EchantillonTemperature dans le fichier ini", err)

	PortWeb, err = c.String("web.PortWeb")
	gestionErr("Impossible de lire valeur web.PortWeb dans le fichier ini", err)

	Plage1, err = c.String("plages.Plage1")
	gestionErr("Impossible de lire valeur plages.Plage1 dans le fichier ini", err)

	Plage2, err = c.String("plages.Plage2")
	gestionErr("Impossible de lire valeur plages.Plage2 dans le fichier ini", err)

	Plage3, err = c.String("plages.Plage3")
	gestionErr("Impossible de lire valeur plages.Plage3 dans le fichier ini", err)

	TPlage1, _ = jodaTime.Parse("HH:mm", Plage1)
	formattedPlage1 = strftime.Format("%H:%M", TPlage1)

	TPlage2, _ = jodaTime.Parse("HH:mm", Plage2)
	formattedPlage2 = strftime.Format("%H:%M", TPlage2)

	TPlage3, _ = jodaTime.Parse("HH:mm", Plage3)
	formattedPlage3 = strftime.Format("%H:%M", TPlage3)

	// Graphana
	grafanaActif, err = c.Bool("grafana.actif")
	gestionErr("Impossible de lire valeur grafana.actif dans le fichier ini", err)

	graphiteHost, err = c.String("grafana.Adresse")
	gestionErr("Impossible de lire valeur grafana.Adresse dans le fichier ini", err)

	graphitePort, err = c.Int("grafana.Port")
	gestionErr("Impossible de lire valeur grafana.Port dans le fichier ini", err)

	// Mqtt
	serveurMqtt, err = c.String("mqtt.Serveur")
	gestionErr("Impossible de lire valeur mqtt.Serveur dans le fichier ini", err)

	portMqtt, err = c.String("mqtt.Port")
	gestionErr("Impossible de lire valeur mqtt.Port dans le fichier ini", err)

	// Topics
	salonTemperature, err = c.String("topics.salonTemperature")
	gestionErr("Impossible de lire valeur topics.salonTemperature dans le fichier ini", err)

	salonHumidite, err = c.String("topics.salonHumidite")
	gestionErr("Impossible de lire valeur topics.salonHumidite dans le fichier ini", err)

	cmdPortail, err = c.String("topics.cmdPortail")
	gestionErr("Impossible de lire valeur topics.cmdPortail dans le fichier ini", err)

	etatPortail, err = c.String("topics.etatPortail")
	gestionErr("Impossible de lire valeur topics.etatPortail dans le fichier ini", err)

	// Temperature minimum electrolyse
	TemperatureElectrolyseInt, err = c.Int("temperatureElectrolyse.Temperature")
	gestionErr("Impossible de lire valeur temperatureElectrolyse.Temperature dans le fichier ini", err)

	// Urls
	lampePortail, err = c.String("url.lampePortail")
	gestionErr("Impossible de lire valeur url.lampePortail dans le fichier ini", err)

	// Telegram les identifiants
	telegramToken, err = c.String("telegram.token")
	gestionErr("Impossible de lire valeur telegram.token dans le fichier ini", err)

	telegramId, err = c.Int("telegram.id")
	gestionErr("Impossible de lire valeur telegram.id dans le fichier ini", err)

	// Teleinfo
	teleinfoActif, err = c.Bool("teleinfo.actif")
	gestionErr("Impossible de lire valeur teleinfo.actif dans le fichier ini", err)

	teleinfoString, err = c.String("teleinfo.lecture")
	gestionErr("Impossible de lire valeur teleinfo.lecture dans le fichier ini", err)

	// Metrics Raspberry
	dureeMetrcisString, err = c.String("metrics.duree")
	gestionErr("Impossible de lire valeur metrics.duree dans le fichier ini", err)

	//Iftt
	ifttKey, err = c.String("iftt.key")
	gestionErr("Impossible de lire valeur iftt.key dans le fichier ini", err)

	// Graphana
	netatmoActif, err = c.Bool("netatmo.actif")
	gestionErr("Impossible de lire valeur netatmo.actif dans le fichier ini", err)

	// Write fichier duration filtration
	dureeWriteFichier, err = c.String("writeFichier.timeFichier")
	gestionErr("Impossible de lire valeur writeFichier.timeFichier dans le fichier ini", err)

	cheminWriteFichier, err = c.String("writeFichier.cheminFichier")
	gestionErr("Impossible de lire valeur writeFichier.cheminFichier dans le fichier ini", err)

	//-- Live Objects ------------------------------------------------------------------
	cleApiLiveObjects, err = c.String("liveobjects.cleApi")
	gestionErr("Impossible de lire valeur liveobjects.cleApi dans le fichier ini", err)

	urlLiveObjects, err = c.String("liveobjects.url")
	gestionErr("Impossible de lire valeur liveobjects.url dans le fichier ini", err)

	topicPushLiveObjects, err = c.String("liveobjects.url")
	gestionErr("Impossible de lire valeur liveobjects.url dans le fichier ini", err)

	// Convertion des temps ------------------------------------------------------
	s := strings.Split(FrequenceMesuresString, " ")
	duree, temps := s[0], s[1]
	dureeInt, _ := strconv.Atoi(duree)
	FrequenceMesuresTimeDuration = ParseTemps(temps, dureeInt)

	s = strings.Split(FrequenceLedString, " ")
	duree, temps = s[0], s[1]
	dureeInt, _ = strconv.Atoi(duree)
	FrequenceLedTimeDuration = ParseTemps(temps, dureeInt)

	s = strings.Split(teleinfoString, " ")
	duree, temps = s[0], s[1]
	dureeInt, _ = strconv.Atoi(duree)
	TeleinfoLectureTimeDuration = ParseTemps(temps, dureeInt)

	s = strings.Split(dureeMetrcisString, " ")
	duree, temps = s[0], s[1]
	dureeInt, _ = strconv.Atoi(duree)
	dureeMetrcisInt = ParseTemps(temps, dureeInt)

	s = strings.Split(dureeWriteFichier, " ")
	duree, temps = s[0], s[1]
	dureeInt, _ = strconv.Atoi(duree)
	dureeWriteFichierInt = ParseTemps(temps, dureeInt)
}

//---------------------------------------------------------------------------
// Commande IFTT
// Entrées : event
// Sorties :
//---------------------------------------------------------------------------
func iftt(event string) {
	url := "https://maker.ifttt.com/trigger/" + event + "/with/key/" + ifttKey

	req, err := http.NewRequest("POST", url, strings.NewReader(""))
	gestionErr("Impossible de créer la requête IFTT", err)

	client := &http.Client{}
	resp, err := client.Do(req)
	gestionErr("Impossible de faire la requête IFTT", err)
	defer resp.Body.Close()

	//fmt.Println("response Status:", resp.Status)
	//fmt.Println("response Headers:", resp.Header)
}

//---------------------------------------------------------------------------
// Lecture clavier
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func clavier() {
	for {
		reader := bufio.NewReader(os.Stdin)
		char, _, err := reader.ReadRune()

		gestionErr("Impossible d'interroger le clavier", err)

		//fmt.Println(char)

		dateTime = jodaTime.Format("dd/MM/YYY HH:mm:ss", time.Now())

		switch strings.ToUpper(string(char)) {

		case "H":
			fmt.Println("Aide sur les commandes :")
			fmt.Println("")
			fmt.Println("A : Lampe portail")
			fmt.Println("P : Pompe piscine")
			fmt.Println("E : Electrolyse")
			fmt.Println("L : lampes extérieures")
			fmt.Println("T : Températures")
			fmt.Println("Q : Quitter le programme")
			break

		// Permet d'allumer et éteindre la pompe
		case "P":
			if etatPompe == 0 {
				pompeOn()
			} else {
				pompeOff()
				electrolyseOff()
			}
			mapDate["Pompe"] = dateTime
			break

		// Permet d'allumer et éteindre l'electrolyse
		case "E":
			if etatElectrolyse == 0 {
				if etatPompe == 1 {
					electrolyseOn()
				}
			} else {
				electrolyseOff()
			}
			mapDate["Electrolyse"] = dateTime
			break

		// Permet d'allumer et éteindre les lampes de dehors du PoolHouse
		case "L":
			if etatLampe == 0 {
				etatLampe = 1
				pinLampe.High()
				pushbulletSend("Domos", "Lampe On")

			} else {
				etatLampe = 0
				pinLampe.Low()
				pushbulletSend("Domos", "Lampe Off")

			}
			mapDate["Lampe"] = dateTime
			break

		// Permet de visualiser la température et durée de filtration
		case "T":
			fmt.Println("---------------------------------------------------------")
			fmt.Println(fmt.Sprintf("Température eau.........: %.2f °c", tEauFloat))
			fmt.Println(fmt.Sprintf("Température air.........: %.2f °c", tAirFloat))
			fmt.Println(fmt.Sprintf("Température air local...: %.2f °c", tAirLocalFloat))
			fmt.Println(fmt.Sprintf("Durée filtration........: %.2f heures", dureeFiltration))
			fmt.Println(fmt.Sprintf("Plage 1.................: %s à %s", formattedPlage1, formattedPlage1Fin))
			fmt.Println(fmt.Sprintf("Plage 2.................: %s à %s", formattedPlage2, formattedPlage2Fin))
			fmt.Println(fmt.Sprintf("Plage 3.................: %s à %s", formattedPlage3, formattedPlage3Fin))
			fmt.Print("Durée de fonctionnement.: ")
			fmt.Println(ms.t)
			fmt.Println("---------------------------------------------------------")
			break

		case "A":
			if etatLampePortail == 0 {
				fmt.Println(" - Allumage lampe portail")
				//http.Get(lampePortail + "inter1_on")
				publishMqtt("cmnd/sonoff/power", "on")
				pushbulletSend("Domos", "Lampe portail On")
				etatLampePortail = 1
			} else {
				fmt.Println(" - Extinction lampe portail")
				//http.Get(lampePortail + "inter1_off")
				pushbulletSend("Domos", "Lampe portail Off")
				publishMqtt("cmnd/sonoff/power", "off")
				etatLampePortail = 0
			}
			mapDate["LampePortail"] = dateTime
			break

		// Permet de quitter proprement le programme
		case "Q":
			pompeOff()
			electrolyseOff()
			pinLampe.Low()
			pinLampePiscine.Low()
			pinLed.Low()
			WriteFichier(ms.compteur)
			os.Exit(1)
			break
		}
	}
}

//---------------------------------------------------------------------------
// Calcul la durée de filtration
// Entrées :
// Sorties : dureeTempo (float64)
//---------------------------------------------------------------------------
func calculDuree() float64 {
	// Calcul de la durée de filtration
	dureeTempo := tEauFloat / 2
	dureeTempo = Round(dureeTempo / 3)
	return dureeTempo
}

//---------------------------------------------------------------------------
// Fonctions On Off des GPIO
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func pompeOn() {
	dateTime = jodaTime.Format("dd/MM/YYY HH:mm:ss", time.Now())
	etatPompe = 1
	pinPompe.High()
	texte = fmt.Sprintf("- Pompe On : %.2f°C", tEauFloat)
	Telegram(texte, telegramToken, telegramId)
	fmt.Println("- Pompe ON  : " + dateTime)
	mapDate["Pompe"] = dateTime
	ms.Flag = true
	data = "Domos.poolhouse.localTechnique.Pompe"
	value = strconv.Itoa(etatPompe)
	err := GraphiteCnx(data, value, graphiteHost, graphitePort)
	errGraphique(err, "9")
}

func pompeOff() {
	dateTime = jodaTime.Format("dd/MM/YYY HH:mm:ss", time.Now())
	etatPompe = 0
	pinPompe.Low()
	texte = fmt.Sprintf("- Pompe Off : %.2f°C", tEauFloat)
	Telegram(texte, telegramToken, telegramId)
	fmt.Println("- Pompe OFF : " + dateTime)
	mapDate["Pompe"] = dateTime
	ms.Flag = false
	data = "Domos.poolhouse.localTechnique.Pompe"
	value = strconv.Itoa(etatPompe)
	err := GraphiteCnx(data, value, graphiteHost, graphitePort)
	errGraphique(err, "10")
}

func electrolyseOn() {
	dateTime = jodaTime.Format("dd/MM/YYY HH:mm:ss", time.Now())
	if etatPompe == 1 && tEauFloat > float64(TemperatureElectrolyseInt) {
		etatElectrolyse = 1
		pinElectrolyse.High()
		fmt.Println("- Electrolyse ON  : " + dateTime)
		mapDate["Electrolyse"] = dateTime
		data = "Domos.poolhouse.localTechnique.Electrolyse"
		value = strconv.Itoa(etatElectrolyse)
		err := GraphiteCnx(data, value, graphiteHost, graphitePort)
		errGraphique(err, "11")
	}
}

func electrolyseOff() {
	dateTime = jodaTime.Format("dd/MM/YYY HH:mm:ss", time.Now())
	etatElectrolyse = 0
	pinElectrolyse.Low()
	fmt.Println("- Electrolyse OFF  : " + dateTime)
	mapDate["Electrolyse"] = dateTime
	data = "Domos.poolhouse.localTechnique.Electrolyse"
	value = strconv.Itoa(etatElectrolyse)
	err := GraphiteCnx(data, value, graphiteHost, graphitePort)
	errGraphique(err, "12")
}

// calculPlages : Permet de calculer la durée des plages de filtation
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func calculPlages() {
	TPlage1Fin := TPlage1.Add(time.Duration(dureeFiltration*60) * time.Minute)
	formattedPlage1Fin = strftime.Format("%H:%M", TPlage1Fin)

	TPlage2Fin := TPlage2.Add(time.Duration(dureeFiltration*60) * time.Minute)
	formattedPlage2Fin = strftime.Format("%H:%M", TPlage2Fin)

	TPlage3Fin := TPlage3.Add(time.Duration(dureeFiltration*60) * time.Minute)
	formattedPlage3Fin = strftime.Format("%H:%M", TPlage3Fin)
}

// mqttSubscribe : Permet de s'abonner à MQTT
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func mqttSubscribe() {
	//opts := mqtt.NewClientOptions().AddBroker("tcp://" + serveurMqtt + ":" + portMqtt).SetClientID("domos")
	opts := mqtt.NewClientOptions().AddBroker("tcp://192.168.1.19:1883").SetClientID("domos")
	opts.SetKeepAlive(2 * time.Second)
	opts.SetDefaultPublishHandler(f)
	opts.SetPingTimeout(1 * time.Second)

	c = mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	if token := c.Subscribe("#", 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
}

//---------------------------------------------------------------------------
// Permet d'écouter MQTT
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	//fmt.Printf("TOPIC: %s\n", msg.Topic())
	//fmt.Printf("MSG: %s\n", msg.Payload())

	switch msg.Topic() {

	case salonTemperature:
		_ = json.Unmarshal([]byte(msg.Payload()), &datamqtt)
		data = "Domos.maison.salon.temperature"
		//fmt.Println(datamqtt["valeur"])
		valeur := fmt.Sprintf("%v", datamqtt["valeur"])
		err := GraphiteCnx(data, valeur, graphiteHost, graphitePort)
		gestionErr("Erreur d'insertion dans Mqtt pour : salon temperature", err)

	case salonHumidite:
		_ = json.Unmarshal([]byte(msg.Payload()), &datamqtt)
		data = "Domos.maison.salon.humidite"
		//fmt.Println(datamqtt["valeur"])
		valeur := fmt.Sprintf("%v", datamqtt["valeur"])
		err := GraphiteCnx(data, valeur, graphiteHost, graphitePort)
		gestionErr("Erreur d'insertion dans Graphite pour : salon humidite", err)

	case etatPortail:
		data = "Domos.exterieure.portail.etat"
		_ = json.Unmarshal([]byte(msg.Payload()), &datamqttPortail)
		etat := fmt.Sprintf("%s", datamqttPortail["portail"])
		wifi := fmt.Sprintf("%v", datamqttPortail["wifiRSSI"])
		ouvertures := fmt.Sprintf("%v", datamqttPortail["ouvertures"])
		if etat == "ON" {
			valeur := "1"
			err := GraphiteCnx(data, valeur, graphiteHost, graphitePort)
			gestionErr("Erreur d'insertion dans Graphite pour : etat portail", err)
		} else {
			valeur := "0"
			err := GraphiteCnx(data, valeur, graphiteHost, graphitePort)
			gestionErr("Erreur d'insertion dans Graphite pour : etat portail", err)
		}

		data = "Domos.exterieure.portail.wifi"
		err := GraphiteCnx(data, wifi, graphiteHost, graphitePort)
		gestionErr("Erreur d'insertion dans Graphite pour : signal wifi portail", err)

		data = "Domos.exterieure.portail.ouvertures"
		err = GraphiteCnx(data, ouvertures, graphiteHost, graphitePort)
		gestionErr("Erreur d'insertion dans Graphite pour : nombres ouvertures portail", err)
	}
}

// publishMqtt : Permet de publier sur MQTT
// Entrées : topic string, valeur string
// Sorties :
//---------------------------------------------------------------------------
func publishMqtt(topic string, valeur string) {
	token := c.Publish(topic, 0, false, valeur)
	token.Wait()
}

// lectureFrame : Goroutine - Permet de décoder les frames téléinfo
// Entrées : port (*serial.Port)
// Sorties :
//---------------------------------------------------------------------------
//	PAPP:00180 HHPHC:E ISOUSC:45 HCHP:000650048 IMAX:028 PTEC:HP.. IINST:001 MOTDETAT:000000 ADCO:701601357467 OPTARIF:HC.. HCHC:000343736
//	PAPP     : Puissance apparente : PAPP ( 5 car. unité = Volt.ampères)
//	HHPHC    : Groupe horaire si option = heures creuses ou tempo : HHPHC (1 car.)
//	ISOUSC   : Intensité souscrite : ISOUSC ( 2 car. unité = ampères)
//	HCHP     : Index heures pleines si option = heures creuses : HCHP ( 9 car. unité = Wh)
//	IMAX     : Intensité maximale : IMAX ( 3 car. unité = ampères)
//	PTEC     : Période tarifaire en cours : PTEC ( 4 car.)
//	IINST    : Intensité instantanée : IINST ( 3 car. unité = ampères)
//	MOTDETAT : Mot d’état (autocontrôle) : MOTDETAT (6 car.)
//	ADCO     : N° d’identification du compteur : ADCO (12 caractères)
//	OPTARIF  : Option tarifaire (type d’abonnement) : OPTARIF (4 car.)
//	HCHC     : Index heures creuses si option = heures creuses : HCHC ( 9 car. unité = Wh)
//---------------------------------------------------------------------------
func lectureFrame(port *serial.Port) {
	for {

		//fmt.Println("Port : %s", serialDevice)
		//fmt.Println("Port : %s", port)

		//if err != nil {
		//	glog.Exitf("Error: %s - %s", err, port)
		//}

		reader := NewReader(port)
		frame, err := reader.ReadFrame()

		gestionErr("Erreur de lecture port série", err)

		for key, value := range frame {
			data = "Domos.poolhouse.localTechnique.compteur.edf." + key
			err = GraphiteCnx(data, value, graphiteHost, graphitePort)
			gestionErr("Erreur d'insertion dans graphite", err)

		}
		time.Sleep(TeleinfoLectureTimeDuration * time.Millisecond)
	}
}

// writeFichierDurationFiltration : Permet d'enregistrer le fichier duration filtration
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func writeFichierDurationFiltration() {
	for {
		WriteFichier(ms.compteur)
		time.Sleep(dureeWriteFichierInt * time.Millisecond)
	}
}

// OpenPort : Permet de paramètrer le port série
// Entrées : serialDevice (string)
// Sorties : port (*serial.Port)
//---------------------------------------------------------------------------
func OpenPort(serialDevice string) (*serial.Port, error) {
	cfg := &serial.Config{
		Name:     serialDevice,
		Baud:     1200,
		Size:     7,
		Parity:   serial.ParityEven,
		StopBits: serial.Stop1,
	}
	return serial.OpenPort(cfg)
}

// counter : Compteur de durée de filtration
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func (ms *MyStruct) counter() {
	for {
		if ms.Flag == true {
			ms.compteur++
			//WriteFichier(ms.compteur)
		}
		time.Sleep(1 * time.Second)
		ms.t = time.Duration(ms.compteur) * time.Second
	}
}
