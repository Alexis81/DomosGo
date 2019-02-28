package main

import (
	"bufio"
	"fmt"
	"math"
	"net"
	"os"
	//"reflect"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/marpaia/graphite-golang" // https://github.com/marpaia/graphite-golang
	"github.com/mitsuse/pushbullet-go"
	"github.com/mitsuse/pushbullet-go/requests"
	"github.com/vjeantet/jodaTime" // https://github.com/vjeantet/jodaTime
	"gopkg.in/telegram-bot-api.v4"
)

// Stats : Structure du Json
type Stats struct {
	Type                                                           string `json:"type"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_Cpu             string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Cpu"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemTotal        string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Total"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemAvailable    string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Available"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemUsed         string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Used"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemUsedPourcent string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.UsedPourcent"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemUsedFree     string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.UsedFree"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemUsedActive   string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.UsedActive"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemUsedInactive string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.UsedInactive"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemWired        string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Wired"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemBuffers      string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Buffers"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemCached       string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Cached"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemWriteback    string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Writeback"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemDirty        string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Dirty"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemWritebackTmp string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.WritebackTmp"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemShared       string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Shared"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemSlab         string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Slab"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemPageTables   string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.PageTables"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_MemSwapCached   string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.SwapCached"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_Temperature     string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Temperature"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_Frequence       string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Frequence"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Stats_Elapsed         string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Elapsed"`
	Version                                                        string `json:"version"`
}

// StatsGolang : Structure du Json
type StatsGolang struct {
	Domos_Poolhouse_LocalTechnique_Raspberry_Go_GoRoutines      string `json:"Domos.poolhouse.localTechnique.Raspberry.Go.GoRoutines"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemAlloc        string `json:"Domos.poolhouse.localTechnique.Raspberry.Go.MemAlloc"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemFrees        string `json:"Domos.poolhouse.localTechnique.Raspberry.Go.MemFrees"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemHeapAlloc    string `json:"Domos.poolhouse.localTechnique.Raspberry.Go.MemHeapAlloc"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemHeapIdle     string `json:"Domos.poolhouse.localTechnique.Raspberry.Go.MemHeapIdle"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemHeapInUse    string `json:"Domos.poolhouse.localTechnique.Raspberry.Go.MemHeapInUse"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemHeapObjects  string `json:"Domos.poolhouse.localTechnique.Raspberry.Go.MemHeapObjects"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemHeapSys      string `json:"Domos.poolhouse.localTechnique.Raspberry.Go.MemHeapSys"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemMallocs      string `json:"Domos.poolhouse.localTechnique.Raspberry.Go.MemMallocs"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemNumGc        string `json:"Domos.poolhouse.localTechnique.Raspberry.Go.MemNumGc"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemPauseTotalNs string `json:"Domos.poolhouse.localTechnique.Raspberry.Go.MemPauseTotalNs"`
	Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemSyk          string `json:"Domos.poolhouse.localTechnique.Raspberry.Go.MemSys"`
}

// StatsNetatmo : Structure du Json
type StatsNetatmo struct {
	Type                                       string `json:"type"`
	Domos_Poolhouse_Bureau_Netatmo_Co2         string `json:"Domos.poolhouse.bureau.netatmo.Co2"`
	Domos_Poolhouse_Bureau_Netatmo_Humidity    string `json:"Domos.poolhouse.bureau.netatmo.Humidity"`
	Domos_Poolhouse_Bureau_Netatmo_Noise       string `json:"Domos.poolhouse.bureau.netatmo.Noise"`
	Domos_Poolhouse_Bureau_Netatmo_Pressure    string `json:"Domos.poolhouse.bureau.netatmo.Pressure"`
	Domos_Poolhouse_Bureau_Netatmo_Temperature string `json:"Domos.poolhouse.bureau.netatmo.Temperature"`
	Domos_Poolhouse_Bureau_Netatmo_Elapsed     string `json:"Domos.poolhouse.bureau.netatmo.Elapsed"`
	Version                                    string `json:"version"`
}

// Round : Permet d'arrondir un float64
//---------------------------------------------------------------------------
func Round(input float64) float64 {
	if input < 0 {
		return math.Ceil(input - 0.5)
	}
	return math.Floor(input + 0.5)
}

// GraphiteCnx : Connexion au serveur graphite
//---------------------------------------------------------------------------
func GraphiteCnx(data string, value string, graphiteHost string, graphitePort int) error {

	/*
		fmt.Print("Data : ")
		fmt.Println(data)
		fmt.Print("Value : ")
		fmt.Println(value)
		fmt.Print("Host : ")
		fmt.Println(graphiteHost)
		fmt.Print("Port : ")
		fmt.Println(graphitePort)
	*/

	var err error

	if grafanaActif {
		//fmt.Println("Graphite OK")
	recommence:

		gr, err := graphite.NewGraphite(graphiteHost, graphitePort)
		gestionErr("Impossible de se connecter à Graphite", err)

		if err != nil {
			dateTime := jodaTime.Format("dd/MM/YYY HH:mm:ss", time.Now())
			fmt.Println(dateTime + "Connexion Graphite KO")
			time.Sleep(5 * time.Second)
			goto recommence
		}

		/*
			fmt.Print("Fonction Graphite : ")
			fmt.Print(data)
			fmt.Print(" - ")
			fmt.Println(value)
		*/

		err = gr.SimpleSend(data, value)
		gestionErr("Impossible d'écrire les données dans Graphite", err)

		err = gr.Disconnect()
		gestionErr("Impossible de se déconnecter de Graphite", err)

	}
	return err

}

//---------------------------------------------------------------------------
// Permet d'envoyer des messages PushBullet
// Entrées : tilte (string) - bosy (string)
// Sorties :
//---------------------------------------------------------------------------
func pushbulletSend(title string, body string) {
	// Set the access token.
	token := "o.eMRgrmveJtS4SUKAlIsIHBHIxS46RYdv"

	// Create a client for Pushbullet.
	pb := pushbullet.New(token)

	// Create a push. The following codes create a note, which is one of push types.
	n := requests.NewNote()
	n.Title = title
	n.Body = body

	// Send the note via Pushbullet.
	if _, err := pb.PostPushesNote(n); err != nil {
		//fmt.Printf(os.Stderr, "error: %s\n", err)
		return
	}
}

// Telegram : Permet d'envoyer des messages dans Telegram
// Entrées : message string
// Sorties :
//---------------------------------------------------------------------------
func Telegram(message string, telegramToken string, telegramId int) {

	bot, err := tgbotapi.NewBotAPI(telegramToken)
	gestionErr("Telegram : problème de Token", err)

	msg := tgbotapi.NewMessage(int64(telegramId), message)
	msg.ParseMode = "markdown"
	_, err = bot.Send(msg)
	gestionErr("Telegram : problème lors de l'envoi du message", err)
}

// Ipadresse : Trouve adresse IP du Raspberry
//---------------------------------------------------------------------------
func Ipadresse() string {
	netInterfaceAddresses, err := net.InterfaceAddrs()

	if err != nil {
		return ""
	}

	for _, netInterfaceAddress := range netInterfaceAddresses {
		networkIp, ok := netInterfaceAddress.(*net.IPNet)
		if ok && !networkIp.IP.IsLoopback() && networkIp.IP.To4() != nil {
			ip := networkIp.IP.String()
			//fmt.Println("Resolved Host IP: " + ip)
			return ip
		}
	}
	return ""
}

// ParseTemps : Permet de parser les temps
// Entrées : temps string, duree time.Duration
// Sorties : time.Duration
//---------------------------------------------------------------------------
func ParseTemps(temps string, duree int) time.Duration {

	//fmt.Print(temps)
	//fmt.Println(duree)

	switch strings.ToLower(temps) {

	case "ms", "milliseconde", "millisecondes":
		duree = duree * 1
		break
	case "s", "seconde", "secondes":
		duree = duree * 1000
		break
	case "m", "minute", "minutes":
		duree = duree * 60000
	case "h", "heure", "heures":
		duree = duree * 3600000
		break
	default:
		fmt.Println("Erreur dans la convertion des temps du fichier config.ini")

	}
	return time.Duration(duree)
}

// WriteFichier : Permet d'écrire dans un fichier
// Entrées : donnees (int64)
// Sorties :
//---------------------------------------------------------------------------
func WriteFichier(donnees int64) {
	fileHandle, err := os.Create(cheminWriteFichier)
	gestionErr("Erreur de création du fichier", err)

	writer := bufio.NewWriter(fileHandle)
	defer fileHandle.Close()
	fmt.Fprintf(writer, strconv.FormatInt(donnees, 10))
	writer.Flush()
}

// LireFichier : Permet de lire un fichier
// Entrées : chemin (string)
// Sorties : vtexte (int64) - vtime (time.duration)
//---------------------------------------------------------------------------
func LireFichier(chemin string) (int64, time.Duration) {

	var vtexte int64
	var vtime time.Duration
	var err error
	var file, _ = os.Open(chemin)
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		vtexte, err = strconv.ParseInt(line, 10, 64)
		//fmt.Println(vtexte)
		if err != nil {
			panic(err)
		}
		vtime = time.Duration(vtexte) * time.Second
	}
	return vtexte, vtime
}

//---------------------------------------------------------------------------
// Permet de grapher les métriques dur Raspberry PI 3
// Entrées : chemin (int64)
// Sorties : vtexte (int64) - vtime (time.duration)
//---------------------------------------------------------------------------
func statCPU() {
	stats := Stats{}
	for {
		//--- API STATS RASPBERRY-------------------------------------------------------------------------------
		// Lecture des sondes
		rs, err := http.Get("http://192.168.1.22:1001/stats")
		// Process response
		if err != nil {
			panic(err) // More idiomatic way would be to print the error and die unless it's a serious error
		}
		defer rs.Body.Close()
		if rs.StatusCode == 200 {
			temp, _ := ioutil.ReadAll(rs.Body)
			err := json.Unmarshal(temp, &stats)
			if err != nil {
				fmt.Println("There was an error:", err)
			}
			e := reflect.ValueOf(&stats).Elem()
			for i := 0; i < e.NumField(); i++ {
				if !(strings.Contains(e.Type().Field(i).Name, "Type") || strings.Contains(e.Type().Field(i).Name, "Version")) {

					//fmt.Println(" " + f.Interface().(string) + " = " + o.Interface().(string))
					err = GraphiteCnx(strings.Replace(e.Type().Field(i).Name, "_", ".", -1), e.Field(i).Interface().(string), graphiteHost, graphitePort)
					if err != nil {
						fmt.Println("Erreur insertion dans graphite18")
					}
				}
			}
		}
		time.Sleep(dureeMetrcisInt * time.Millisecond)
	}
}

// Netatmo : Permet de grapher les métriques de Netatmo
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func Netatmo() {

	statsNetatmo := StatsNetatmo{}

	for {
		//--- API STATS RASPBERRY-------------------------------------------------------------------------------
		// Lecture des sondes
		rs, err := http.Get("http://192.168.1.22:1003/statsNetatmo")
		// Process response
		if err != nil {
			panic(err) // More idiomatic way would be to print the error and die unless it's a serious error
		}

		//fmt.Println(rs.Header)

		defer rs.Body.Close()

		if rs.StatusCode == 200 {

			temp, _ := ioutil.ReadAll(rs.Body)

			err := json.Unmarshal(temp, &statsNetatmo)
			if err != nil {
				fmt.Println("There was an error:", err)
			}

			e := reflect.ValueOf(&statsNetatmo).Elem()

			for i := 0; i < e.NumField(); i++ {
				if !(strings.Contains(e.Type().Field(i).Name, "Type") || strings.Contains(e.Type().Field(i).Name, "Version")) {

					//fmt.Println(" " + strings.Replace(e.Type().Field(i).Name, "_", ".", -1) + " = " + e.Field(i).Interface().(string))
					err = GraphiteCnx(strings.Replace(e.Type().Field(i).Name, "_", ".", -1), e.Field(i).Interface().(string), graphiteHost, graphitePort)
					if err != nil {
						fmt.Println("Erreur insertion dans graphite18")
					}
				}
			}
		}
		time.Sleep(FrequenceMesuresTimeDuration * time.Millisecond)
	}
}

//---------------------------------------------------------------------------
// Permet de pousser les métriques Goland dans Graphite
// Entrées :
// Sorties :
//---------------------------------------------------------------------------
func metriquesGolang() {
	statsGolang := StatsGolang{}
	for {
		var gc float64
		var oldGc float64

		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		//total time that the garbage collector has paused the program
		gc = float64(memStats.PauseTotalNs) / float64(time.Millisecond/time.Nanosecond)
		gctime := (gc - oldGc) / float64(time.Millisecond/time.Nanosecond)
		Gcgarbadge := fmt.Sprintf("%v", gctime)
		oldGc = gc

		// Enrichir la structure
		statsGolang.Domos_Poolhouse_LocalTechnique_Raspberry_Go_GoRoutines = strconv.Itoa(runtime.NumGoroutine())
		statsGolang.Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemAlloc = fmt.Sprintf("%v", memStats.Alloc)
		statsGolang.Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemFrees = fmt.Sprintf("%v", memStats.Frees)
		statsGolang.Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemHeapAlloc = fmt.Sprintf("%v", memStats.HeapAlloc)
		statsGolang.Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemHeapIdle = fmt.Sprintf("%v", memStats.HeapIdle)
		statsGolang.Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemHeapInUse = fmt.Sprintf("%v", memStats.HeapInuse)
		statsGolang.Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemHeapObjects = fmt.Sprintf("%v", memStats.HeapObjects)
		statsGolang.Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemHeapSys = fmt.Sprintf("%v", memStats.HeapSys)
		statsGolang.Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemMallocs = fmt.Sprintf("%v", memStats.Mallocs)
		statsGolang.Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemNumGc = fmt.Sprintf("%v", memStats.NumGC)
		statsGolang.Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemPauseTotalNs = Gcgarbadge
		statsGolang.Domos_Poolhouse_LocalTechnique_Raspberry_Go_MemSyk = fmt.Sprintf("%v", memStats.Sys)

		e := reflect.ValueOf(&statsGolang).Elem()

		for i := 0; i < e.NumField(); i++ {
			//fmt.Println(" " + strings.Replace(e.Type().Field(i).Name, "_", ".", -1) + " = " + e.Field(i).Interface().(string))
			err := GraphiteCnx(strings.Replace(e.Type().Field(i).Name, "_", ".", -1), e.Field(i).Interface().(string), graphiteHost, graphitePort)
			if err != nil {
				fmt.Println("Erreur insertion dans graphite18")
			}
		}
		time.Sleep(dureeMetrcisInt * time.Millisecond)
	}
}

//---------------------------------------------------------------------------
// Permet de gérer les erreurs et d'afficher un message
// Entrées : texte string, err error
// Sorties :
//---------------------------------------------------------------------------
func gestionErr(texte string, err error) {
	if err != nil {
		fmt.Println(texte)
		fmt.Println(err)
	}
}
