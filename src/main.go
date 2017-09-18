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
//---------------------------------------------------------------------------

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	//"reflect"
	"flag"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/glog"
	"github.com/hhkbp2/go-strftime"
	"github.com/stianeikeland/go-rpio"
	"github.com/tarm/serial"
	"github.com/vjeantet/jodaTime" // https://github.com/vjeantet/jodaTime
	"github.com/yryz/ds18b20"
	"github.com/zpatrick/go-config"
)

var (
	// Pin pour Led de vie
	pin_Led                  = rpio.Pin(24)
	pin_Pompe                = rpio.Pin(25)
	pin_Electrolyse          = rpio.Pin(18)
	pin_Lampe                = rpio.Pin(22)
	pin_Lampe_Piscine        = rpio.Pin(17)
	DS_Air                   = ""
	DS_Eau                   = ""
	Frequence_Led_String     = ""
	Frequence_Mesures_String = ""
	Port_Web                 = ""
	Plage1                   = ""
	Plage2                   = ""
	Plage3                   = ""
	etat_Pompe               = 0
	etat_Electrolyse         = 0
	etat_Lampe               = 0
	etat_Lampe_Portail       = 0
	input                    = ""
	flag_Plage1              = true
	flag_Plage2              = true
	flag_Plage3              = true
	serveurMqtt              = ""
	portMqtt                 = ""
	salonTemperature         = ""
	salonHumidite            = ""
	cmdPortail               = ""
	etatPortail              = ""
	lampePortail             = ""
	graphiteHost             = "92.154.38.115"
	graphitePort             = 2003
	temperature_eau_string   = ""
	temperature_air_string   = ""
	texte                    = ""
	telegramToken            = ""
	telegramId               = 0
	teleinfoString           = ""
	t_eau                    float64
	Temperature_Electrolyse  int
	t_air                    float64
	err                      error
	data                     string
	value                    string
	Mesure_Temperatures_uint uint
	duree                    float64
	TPlage1                  time.Time
	TPlage2                  time.Time
	TPlage3                  time.Time
	dateTime                 string
	formatted_Plage1         string
	formatted_Plage2         string
	formatted_Plage3         string
	formatted_Plage1_Fin     string
	formatted_Plage2_Fin     string
	formatted_Plage3_Fin     string
	datamqtt                 map[string]interface{}
	c                        mqtt.Client
	Frequence_Mesures_Int    time.Duration
	Frequence_Led_Int        time.Duration
	TeleinfolectureInt       time.Duration
	serialDevice             string
	exporterName             string
	mapDate                  = make(map[string]string)
	dureeMetrcisString       = ""
	dureeMetrcisInt          time.Duration
)

// Structure pour le compteur de durée de filtration
type MyStruct struct {
	Flag     bool
	compteur int64
	t        time.Duration
}

var ms = MyStruct{}

// Fonction pour initialiser le programme
func init() {

	nprocess := runtime.GOMAXPROCS(runtime.NumCPU())
	dateTime = jodaTime.Format("dd/MM/YYY HH:mm:ss", time.Now())

	ms.compteur, ms.t = LireFichier("/root/donnees/dureeFiltration.txt")

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

	// Lecture du fichier Conf/config.ini
	fmt.Println(" - Lecture du fichier config.ini")
	lectureIni()

	// Paramètres pour la communication série pour Téléinfo EDF
	flag.StringVar(&serialDevice, "device", "/dev/ttyS0", "Serial port to read frames from")
	flag.Parse()
	port, _ := OpenPort(serialDevice)

	// Lecture du fichier Conf/config.ini
	fmt.Println(" - Lancement routine Téléinfo EDF")
	go lectureFrame(port)

	// Envoier les stats de Raspberry PI 3
	go statCpu()

	// Se connecter au serveur MQTT
	fmt.Println(" - Abonnements au serveur MQTT")
	mqttSubscribe()

	// Etre certains que la lampe portail soit sur OFF
	etat_Lampe_Portail = 0
	http.Get(lampePortail + "inter1_off")

	// Nombres de processeurs
	fmt.Print(" - Nombres de processeurs : ")
	fmt.Println(nprocess)

}

//---------------------------------------------------------------------------
// Fonction principal du programme
//---------------------------------------------------------------------------
func main() {

	// Démarre le compteur pour la durée de filtration de la piscine
	go ms.counter()

	// Lancement lecture calvier en goroutine
	fmt.Println(" - Scan du clavier")
	go clavier()

	fmt.Println(" - Affectation des GPIO")
	// Open and map memory to access gpio, check for errors
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Unmap gpio memory when done
	defer rpio.Close()

	// Affecte les pins en sortie
	pin_Led.Output()
	pin_Pompe.Output()
	pin_Electrolyse.Output()
	pin_Lampe.Output()

	// Passe les pins à Off
	pin_Pompe.Low()
	pin_Electrolyse.Low()
	pin_Lampe.Low()

	fmt.Println(" - Lancement des Jobs")

	// Lance la lecture des sondes
	go LectureSondes()

	// Permet de faire clignter la led de vie
	go BlinkLed()

	// Create a mux for routing incoming requests
	m := http.NewServeMux()

	//fs := http.FileServer(http.Dir("html"))
	m.HandleFunc("/html/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.URL.Path[1:])
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	// All URLs will be handled by this function
	m.HandleFunc("/toto", func(w http.ResponseWriter, r *http.Request) {
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
			"</div>", t_eau, t_air)
	})

	m.HandleFunc("/temps", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Temp&eacute;rature eau : %.2f°C</li>"+
			"<li>Temp&eacute;rature air : %.2f°C</li>"+
			"<li>Etat pompe eau         : %d - (%s)</li>"+
			"<li>Etat electrolyse       : %d - (%s)</li>"+
			"<li>Etat Lampe             : %d - (%s)</li>"+
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
			"</ul>", t_eau, t_air, etat_Pompe, mapDate["Pompe"], etat_Electrolyse, mapDate["Electrolyse"], etat_Lampe, mapDate["Lampe"], duree, formatted_Plage1, formatted_Plage1_Fin, formatted_Plage2, formatted_Plage2_Fin, formatted_Plage3, formatted_Plage3_Fin, ms.t)
	})

	m.HandleFunc("/pompe/1", func(w http.ResponseWriter, r *http.Request) {
		etat_Pompe = 1
		pin_Pompe.High()
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat pompe eau         : %d</li>"+
			"</ul>", etat_Pompe)
	})

	m.HandleFunc("/pompe/0", func(w http.ResponseWriter, r *http.Request) {
		etat_Pompe = 0
		pin_Pompe.Low()
		etat_Electrolyse = 0
		pin_Electrolyse.Low()
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat pompe eau         : %d</li>"+
			"</ul>", etat_Pompe)
	})

	m.HandleFunc("/electro/1", func(w http.ResponseWriter, r *http.Request) {
		if etat_Pompe == 1 {
			electrolyse_On()
			fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
				"<ul>"+
				"<li>Etat electrolyse         : %d</li>"+
				"</ul>", etat_Electrolyse)
		} else {
			fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
				"<ul>"+
				"<li>Impossible de lancer electrolyse, pompe Off</li>"+
				"</ul>")
		}

	})

	m.HandleFunc("/electro/0", func(w http.ResponseWriter, r *http.Request) {
		etat_Electrolyse = 0
		pin_Electrolyse.Low()
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat electrolyse         : %d</li>"+
			"</ul>", etat_Electrolyse)
	})

	m.HandleFunc("/lampe/0", func(w http.ResponseWriter, r *http.Request) {
		etat_Lampe = 0
		pin_Lampe.Low()
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat lampes         : %d</li>"+
			"</ul>", etat_Lampe)
	})

	m.HandleFunc("/lampe/1", func(w http.ResponseWriter, r *http.Request) {
		etat_Lampe = 1
		pin_Lampe.High()
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat lampes         : %d</li>"+
			"</ul>", etat_Lampe)
	})

	m.HandleFunc("/lampePortail/0", func(w http.ResponseWriter, r *http.Request) {
		etat_Lampe_Portail = 0
		http.Get(lampePortail + "inter1_off")
		pushbulletSend("Domos", "Lampe portail Off")
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat lampes portail        : %d</li>"+
			"</ul>", etat_Lampe_Portail)
	})

	m.HandleFunc("/lampePortail/1", func(w http.ResponseWriter, r *http.Request) {
		etat_Lampe_Portail = 1
		http.Get(lampePortail + "inter1_on")
		pushbulletSend("Domos", "Lampe portail On")
		fmt.Fprintf(w, "<h1><strong>Domos by Golang</strong></h1>"+
			"<ul>"+
			"<li>Etat lampes portail        : %d</li>"+
			"</ul>", etat_Lampe_Portail)
	})

	// Create a server listening on port 8000
	t := &http.Server{
		Addr:    ":" + Port_Web,
		Handler: m,
	}

	ipadresse := Ipadresse()

	fmt.Println(" - Lancement du serveur Web : http://" + ipadresse + ":" + Port_Web)
	// Continue to process new requests until an error occurs
	go t.ListenAndServe()

	//Affiche durée totale de filtartion
	fmt.Print(" - Durée totale de filtartion : ")
	fmt.Println(ms.t)

	fmt.Println(" - Run du programme...")

	// Boucle sans fin ------------------------------------------------------
	for {

		heure_courante := strftime.Format("%H:%M", time.Now())
		//fmt.Println(heure_courante)

		// Calucl la durée des plages
		calcul_Plages()

		if heure_courante > formatted_Plage1 && heure_courante < formatted_Plage1_Fin && flag_Plage1 {
			fmt.Println("- Plage 1")
			pompe_On()
			electrolyse_On()
			pushbulletSend("Domos", "Run Plage 1 : "+temperature_eau_string+" °c")

			flag_Plage1 = false
			flag_Plage2 = true
			flag_Plage3 = true
		}

		if heure_courante > formatted_Plage1_Fin && !flag_Plage1 {
			fmt.Println("- Plage 1 fin")
			pompe_Off()
			electrolyse_Off()
			pushbulletSend("Domos", "Off Plage 1 : "+temperature_eau_string+" °c")

			flag_Plage1 = true
		}

		if heure_courante > formatted_Plage2 && heure_courante < formatted_Plage2_Fin && flag_Plage2 {
			fmt.Println("- Plage 2")
			pompe_On()
			electrolyse_On()
			pushbulletSend("Domos", "Run Plage 2 : "+temperature_eau_string+" °c")

			flag_Plage1 = true
			flag_Plage2 = false
			flag_Plage3 = true
		}

		if heure_courante > formatted_Plage2_Fin && !flag_Plage2 {
			fmt.Println("- Plage 2 fin")
			pompe_Off()
			electrolyse_Off()
			pushbulletSend("Domos", "Off Plage 2 : "+temperature_eau_string+" °c")

			flag_Plage2 = true
		}

		if heure_courante > formatted_Plage3 && heure_courante < formatted_Plage3_Fin && flag_Plage3 {
			fmt.Println("- Plage 3")
			pompe_On()
			electrolyse_On()
			pushbulletSend("Domos", "Run Plage 3 : "+temperature_eau_string+" °c")

			flag_Plage1 = true
			flag_Plage2 = true
			flag_Plage3 = false
		}

		if heure_courante > formatted_Plage3_Fin && !flag_Plage3 {
			fmt.Println("- Plage 3 fin")
			pompe_Off()
			electrolyse_Off()
			pushbulletSend("Domos", "Off Plage 3 : "+temperature_eau_string+" °c")

			flag_Plage3 = true
		}
	}

	time.Sleep(500 * time.Millisecond)

}

//---------------------------------------------------------------------------
// Lire les sondes
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func LectureSondes() {

	for {

		// Lecture des sondes
		t_air, err = ds18b20.Temperature(DS_Air)
		t_eau, err = ds18b20.Temperature(DS_Eau)

		// Température CPU
		re := regexp.MustCompile("\\d+(\\.\\d+)?")
		out, err := exec.Command("cat", "/sys/class/thermal/thermal_zone0/temp").Output()
		if err != nil {
			fmt.Println(err)
		}
		v, err := strconv.ParseFloat(re.FindString(string(out)), 64)
		if err != nil {
			fmt.Println("cmd err")
		}

		// Fréquence CPU
		out, err = exec.Command("cat", "/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_cur_freq").Output()
		if err != nil {
			fmt.Println(err)
		}
		f, err := strconv.ParseFloat(re.FindString(string(out)), 64)
		if err != nil {
			fmt.Println("cmd err")
		}

		// Convertion en String des températures
		temperature_eau_string = strconv.FormatFloat(t_eau, 'f', 2, 64)
		temperature_air_string = strconv.FormatFloat(t_air, 'f', 2, 64)

		duree = calculDuree()

		// Insertion des données dans graphite
		data = "Domos.poolhouse.localTechnique.Air"
		value = temperature_air_string
		err = GraphiteCnx(data, value, graphiteHost, graphitePort)
		if err != nil {
			fmt.Println("Erreur insertion dans graphite")
		}

		data = "Domos.poolhouse.localTechnique.Piscine"
		value = temperature_eau_string
		err = GraphiteCnx(data, value, graphiteHost, graphitePort)
		if err != nil {
			fmt.Println("Erreur insertion dans graphite")
		}

		data = "Domos.poolhouse.localTechnique.Pompe"
		value = strconv.Itoa(etat_Pompe)
		err = GraphiteCnx(data, value, graphiteHost, graphitePort)
		if err != nil {
			fmt.Println("Erreur insertion dans graphite")
		}

		data = "Domos.poolhouse.localTechnique.Electrolyse"
		value = strconv.Itoa(etat_Electrolyse)
		err = GraphiteCnx(data, value, graphiteHost, graphitePort)
		if err != nil {
			fmt.Println("Erreur insertion dans graphite")
		}

		data = "Domos.poolhouse.localTechnique.Lampe"
		value = strconv.Itoa(etat_Lampe)
		err = GraphiteCnx(data, value, graphiteHost, graphitePort)
		if err != nil {
			fmt.Println("Erreur insertion dans graphite")
		}

		data = "Domos.poolhouse.localTechnique.Cpu"
		value = strconv.FormatFloat(v/1000, 'f', 2, 64)
		err = GraphiteCnx(data, value, graphiteHost, graphitePort)
		if err != nil {
			fmt.Println("Erreur insertion dans graphite")
		}

		data = "Domos.poolhouse.localTechnique.Cpu_Frequence"
		value = strconv.FormatFloat(f, 'f', 0, 64)
		err = GraphiteCnx(data, value, graphiteHost, graphitePort)
		if err != nil {
			fmt.Println("Erreur insertion dans graphite")
		}
		time.Sleep(Frequence_Mesures_Int * time.Millisecond)
	}
}

//---------------------------------------------------------------------------
// Faire clignoter la led
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func BlinkLed() {
	for {
		pin_Led.Toggle()
		time.Sleep(Frequence_Led_Int * time.Millisecond)
	}
}

//---------------------------------------------------------------------------
// Permet de lire le fichier ini et placer les valeurs dans des variables
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
	DS_Air, err = c.String("sondes.Air")
	DS_Eau, err = c.String("sondes.Eau")
	Frequence_Led_String, err = c.String("led.Frequence")
	Frequence_Mesures_String, err = c.String("mesures.EchantillonTemperature")

	Port_Web, err = c.String("web.PortWeb")

	Plage1, err = c.String("plages.Plage1")
	Plage2, err = c.String("plages.Plage2")
	Plage3, err = c.String("plages.Plage3")

	TPlage1, _ = jodaTime.Parse("HH:mm", Plage1)
	formatted_Plage1 = strftime.Format("%H:%M", TPlage1)

	TPlage2, _ = jodaTime.Parse("HH:mm", Plage2)
	formatted_Plage2 = strftime.Format("%H:%M", TPlage2)

	TPlage3, _ = jodaTime.Parse("HH:mm", Plage3)
	formatted_Plage3 = strftime.Format("%H:%M", TPlage3)

	// Graphana
	graphiteHost, err = c.String("grafana.Adresse")
	graphitePort, err = c.Int("grafana.Port")

	// Mqtt
	serveurMqtt, err = c.String("mqtt.Serveur")
	portMqtt, err = c.String("mqtt.Port")

	// Topics
	salonTemperature, err = c.String("topics.salonTemperature")
	salonHumidite, err = c.String("topics.salonHumidite")
	cmdPortail, err = c.String("topics.cmdPortail")
	etatPortail, err = c.String("topics.etatPortail")

	// Temperature minimum electrolyse
	Temperature_Electrolyse, err = c.Int("temperatureElectrolyse.Temperature")

	// Urls
	lampePortail, err = c.String("url.lampePortail")

	// Telegram les identifiants
	telegramToken, err = c.String("telegram.token")
	telegramId, err = c.Int("telegram.id")

	// Teleinfo
	teleinfoString, err = c.String("teleinfo.lecture")

	// Metrics Raspberry
	dureeMetrcisString, err = c.String("metrics.duree")

	// Convertion des temps
	s := strings.Split(Frequence_Mesures_String, " ")
	duree, temps := s[0], s[1]
	dureeInt, _ := strconv.Atoi(duree)
	Frequence_Mesures_Int = ParseTemps(temps, dureeInt)

	s = strings.Split(Frequence_Led_String, " ")
	duree, temps = s[0], s[1]
	dureeInt, _ = strconv.Atoi(duree)
	Frequence_Led_Int = ParseTemps(temps, dureeInt)

	s = strings.Split(teleinfoString, " ")
	duree, temps = s[0], s[1]
	dureeInt, _ = strconv.Atoi(duree)
	TeleinfolectureInt = ParseTemps(temps, dureeInt)

	s = strings.Split(dureeMetrcisString, " ")
	duree, temps = s[0], s[1]
	dureeInt, _ = strconv.Atoi(duree)
	dureeMetrcisInt = ParseTemps(temps, dureeInt)
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

		if err != nil {
			fmt.Println(err)
		}

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
			if etat_Pompe == 0 {
				pompe_On()

			} else {
				pompe_Off()
				electrolyse_Off()
			}
			mapDate["Pompe"] = dateTime
			break

		// Permet d'allumer et éteindre l'electrolyse
		case "E":
			if etat_Electrolyse == 0 {
				if etat_Pompe == 1 {
					electrolyse_On()
				}
			} else {
				electrolyse_Off()
			}
			mapDate["Electrolyse"] = dateTime
			break

		// Permet d'allumer et éteindre les lampes de dehors du PoolHouse
		case "L":
			if etat_Lampe == 0 {
				etat_Lampe = 1
				pin_Lampe.High()
				pushbulletSend("Domos", "Lampe On")

			} else {
				etat_Lampe = 0
				pin_Lampe.Low()
				pushbulletSend("Domos", "Lampe Off")

			}
			mapDate["Lampe"] = dateTime
			break

		// Permet de visualiser la température et durée de filtration
		case "T":
			fmt.Println(fmt.Sprintf("Température eau  : %.2f °c", t_eau))
			fmt.Println(fmt.Sprintf("Température air  : %.2f °c", t_air))
			fmt.Println(fmt.Sprintf("Durée filtration : %.2f heures", duree))
			fmt.Println(fmt.Sprintf("Plage 1 : %s à %s", formatted_Plage1, formatted_Plage1_Fin))
			fmt.Println(fmt.Sprintf("Plage 2 : %s à %s", formatted_Plage2, formatted_Plage2_Fin))
			fmt.Println(fmt.Sprintf("Plage 3 : %s à %s", formatted_Plage3, formatted_Plage3_Fin))

			fmt.Print("Durée de fonctionnement : ")
			fmt.Println(ms.t)
			break

		case "A":
			if etat_Lampe_Portail == 0 {
				fmt.Println(" - Allumage lampe portail")
				http.Get(lampePortail + "inter1_on")
				pushbulletSend("Domos", "Lampe portail On")
				etat_Lampe_Portail = 1
			} else {
				fmt.Println(" - Extinction lampe portail")
				http.Get(lampePortail + "inter1_off")
				pushbulletSend("Domos", "Lampe portail Off")
				etat_Lampe_Portail = 0
			}
			mapDate["LampePortail"] = dateTime
			break

		// Permet de quitter proprement le programme
		case "Q":
			pompe_Off()
			electrolyse_Off()
			pin_Lampe.Low()
			pin_Lampe_Piscine.Low()
			pin_Led.Low()
			os.Exit(1)
			break
		}
	}
}

//---------------------------------------------------------------------------
// Calcul la durée de filtration
// Entrées :
// Sorties : duree (float64)
//---------------------------------------------------------------------------
func calculDuree() float64 {
	// Calcul de la durée de filtration
	var duree = t_eau / 2
	duree = Round(duree / 3)

	return duree
}

//---------------------------------------------------------------------------
// Fonctions On Off des GPIO
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func pompe_On() {
	dateTime = jodaTime.Format("dd/MM/YYY HH:mm:ss", time.Now())
	etat_Pompe = 1
	pin_Pompe.High()
	texte = fmt.Sprintf("- Pompe On : %.2f°C", t_eau)
	Telegram(texte, telegramToken, telegramId)
	fmt.Println("- Pompe ON  : " + dateTime)
	mapDate["Pompe"] = dateTime
	ms.Flag = true
	data = "Domos.poolhouse.localTechnique.Pompe"
	value = strconv.Itoa(etat_Pompe)
	err = GraphiteCnx(data, value, graphiteHost, graphitePort)
	if err != nil {
		fmt.Println("Erreur insertion dans graphite")
	}
}

func pompe_Off() {
	dateTime = jodaTime.Format("dd/MM/YYY HH:mm:ss", time.Now())
	etat_Pompe = 0
	pin_Pompe.Low()
	texte = fmt.Sprintf("- Pompe Off : %.2f°C", t_eau)
	Telegram(texte, telegramToken, telegramId)
	fmt.Println("- Pompe OFF : " + dateTime)
	mapDate["Pompe"] = dateTime
	ms.Flag = false
	data = "Domos.poolhouse.localTechnique.Pompe"
	value = strconv.Itoa(etat_Pompe)
	err = GraphiteCnx(data, value, graphiteHost, graphitePort)
	if err != nil {
		fmt.Println("Erreur insertion dans graphite")
	}
}

func electrolyse_On() {
	dateTime = jodaTime.Format("dd/MM/YYY HH:mm:ss", time.Now())
	if etat_Pompe == 1 && t_eau > float64(Temperature_Electrolyse) {
		etat_Electrolyse = 1
		pin_Electrolyse.High()
		fmt.Println("- Electrolyse ON  : " + dateTime)
		mapDate["Electrolyse"] = dateTime
		data = "Domos.poolhouse.localTechnique.Electrolyse"
		value = strconv.Itoa(etat_Electrolyse)
		err = GraphiteCnx(data, value, graphiteHost, graphitePort)
		if err != nil {
			fmt.Println("Erreur insertion dans graphite")
		}
	}
}

func electrolyse_Off() {
	dateTime = jodaTime.Format("dd/MM/YYY HH:mm:ss", time.Now())
	etat_Electrolyse = 0
	pin_Electrolyse.Low()
	fmt.Println("- Electrolyse OFF  : " + dateTime)
	mapDate["Electrolyse"] = dateTime
	data = "Domos.poolhouse.localTechnique.Electrolyse"
	value = strconv.Itoa(etat_Electrolyse)
	err = GraphiteCnx(data, value, graphiteHost, graphitePort)
	if err != nil {
		fmt.Println("Erreur insertion dans graphite")
	}
}

//---------------------------------------------------------------------------
// Permet de calculer la durée des plages de filtation
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func calcul_Plages() {
	TPlage1_Fin := TPlage1.Add(time.Duration(duree*60) * time.Minute)
	formatted_Plage1_Fin = strftime.Format("%H:%M", TPlage1_Fin)

	TPlage2_Fin := TPlage2.Add(time.Duration(duree*60) * time.Minute)
	formatted_Plage2_Fin = strftime.Format("%H:%M", TPlage2_Fin)

	TPlage3_Fin := TPlage3.Add(time.Duration(duree*60) * time.Minute)
	formatted_Plage3_Fin = strftime.Format("%H:%M", TPlage3_Fin)
}

//---------------------------------------------------------------------------
// Permet de s'abonner à MQTT
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
		err = GraphiteCnx(data, valeur, graphiteHost, graphitePort)

	case salonHumidite:
		_ = json.Unmarshal([]byte(msg.Payload()), &datamqtt)
		data = "Domos.maison.salon.humidite"
		//fmt.Println(datamqtt["valeur"])
		valeur := fmt.Sprintf("%v", datamqtt["valeur"])
		err = GraphiteCnx(data, valeur, graphiteHost, graphitePort)

	case etatPortail:
		data = "Domos.exterieure.portail"
		etat := fmt.Sprintf("%s", msg.Payload())
		if etat == "ON1" {
			valeur := "1"
			err = GraphiteCnx(data, valeur, graphiteHost, graphitePort)
		} else {
			valeur := "0"
			err = GraphiteCnx(data, valeur, graphiteHost, graphitePort)
		}
	}
}

//---------------------------------------------------------------------------
// Permet de publier sur MQTT
// Entrées : topic string, valeur string
// Sorties :
//---------------------------------------------------------------------------
func publishMqtt(topic string, valeur string) {
	token := c.Publish(topic, 0, false, valeur)
	token.Wait()
}

//---------------------------------------------------------------------------
// Goroutine - Permet de décoder les frames téléinfo
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

		if err != nil {
			glog.Exitf("Error: %s - %s", err, port)
		}

		reader := NewReader(port)
		frame, err := reader.ReadFrame()

		if err != nil {
			glog.Exitf("Error reading frame from '%s' (%s)\n", serialDevice, err)
		}

		for key, value := range frame {
			data = "Domos.poolhouse.localTechnique.compteur.edf." + key
			err = GraphiteCnx(data, value, graphiteHost, graphitePort)
			//qfmt.Println(key + " --> " + value)
			//fmt.Println(TeleinfolectureInt)
		}

		time.Sleep(TeleinfolectureInt * time.Millisecond)
	}
}

//---------------------------------------------------------------------------
// Permet de paramètrer le port série
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

//---------------------------------------------------------------------------
// Compteur de durée de filtration
// Entrées : serialDevice (string)
// Sorties : port (*serial.Port)
//---------------------------------------------------------------------------
func (ms *MyStruct) counter() {
	for {

		if ms.Flag == true {
			ms.compteur++
			WriteFichier(ms.compteur)
		}

		time.Sleep(1 * time.Second)
		ms.t = time.Duration(ms.compteur) * time.Second
	}
}
