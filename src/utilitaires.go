package main

import (
	"bufio"
	"fmt"
	"math"
	"net"
	"os"
	//"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/marpaia/graphite-golang" // https://github.com/marpaia/graphite-golang
	"github.com/mitsuse/pushbullet-go"
	"github.com/mitsuse/pushbullet-go/requests"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	netio "github.com/shirou/gopsutil/net"
	"gopkg.in/telegram-bot-api.v4"
)

//---------------------------------------------------------------------------
// Permet d'arrondir un float64
//---------------------------------------------------------------------------
func Round(input float64) float64 {
	if input < 0 {
		return math.Ceil(input - 0.5)
	}
	return math.Floor(input + 0.5)
}

//---------------------------------------------------------------------------
// Connexion au serveur graphite
//---------------------------------------------------------------------------
func GraphiteCnx(data string, value string, graphiteHost string, graphitePort int) error {

recommence:

	gr, err := graphite.NewGraphite(graphiteHost, graphitePort)

	if err != nil {
		fmt.Println("Connexion Graphite KO")
		time.Sleep(5 * time.Second)
		goto recommence
	}

	err = gr.SimpleSend(data, value)
	err = gr.Disconnect()

	return err

}

//---------------------------------------------------------------------------
// Trouve adresse IP du Raspberry
//---------------------------------------------------------------------------
func Ipadresse() string {
	netInterfaceAddresses, err := net.InterfaceAddrs()

	//fmt.Println(netInterfaceAddresses)

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
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
}

//---------------------------------------------------------------------------
// Permet de parser les temps
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

//---------------------------------------------------------------------------
// Permet d'envoyer des messages dans Telegram
// Entrées : message string
// Sorties :
//---------------------------------------------------------------------------
func Telegram(message string, telegramToken string, telegramId int) {

	bot, err := tgbotapi.NewBotAPI(telegramToken)

	if err != nil {
		fmt.Println("Telegram : problème de Token")
	}

	msg := tgbotapi.NewMessage(int64(telegramId), message)
	msg.ParseMode = "markdown"
	_, err = bot.Send(msg)

	if err != nil {
		fmt.Println("Telegram : problème de Token")
	}
}

//---------------------------------------------------------------------------
// Permet d'écrire dans un fichier
// Entrées : donnees (int64)
// Sorties :
//---------------------------------------------------------------------------
func WriteFichier(donnees int64) {
	fileHandle, err := os.Create("/root/donnees/dureeFiltration.txt")
	if err != nil {
		fmt.Println("Erreur création du fichier...")
		return
	}

	writer := bufio.NewWriter(fileHandle)
	//fmt.Println("ecriture")

	defer fileHandle.Close()

	fmt.Fprintf(writer, strconv.FormatInt(donnees, 10))

	writer.Flush()
}

//---------------------------------------------------------------------------
// Permet de lire un fichier
// Entrées : chemin (int64)
// Sorties : vtexte (int64) - vtime (time.duration)
//---------------------------------------------------------------------------
func LireFichier(chemin string) (int64, time.Duration) {

	var vtexte int64

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
	}
	vtime := time.Duration(vtexte) * time.Second

	return vtexte, vtime

}

//---------------------------------------------------------------------------
// Permet de grapher les métriques dur Raspberry PI 3
// Entrées : chemin (int64)
// Sorties : vtexte (int64) - vtime (time.duration)
//---------------------------------------------------------------------------
func statCpu() {
	for {
		percentage, _ := cpu.Percent(0, true)
		v, _ := mem.VirtualMemory()

		for idx, cpupercent := range percentage {
			data = "Domos.poolhouse.localTechnique.Raspberry.CPU" + strconv.Itoa(idx)

			value = fmt.Sprintf("%.2f", cpupercent)

			//fmt.Println(reflect.TypeOf(idx))
			//fmt.Println(idx)

			err = GraphiteCnx(data, value, graphiteHost, graphitePort)
			if err != nil {
				fmt.Println("Erreur insertion dans graphite")
			}
		}

		// almost every return value is a struct
		//fmt.Printf("Total: %v, Free:%v, UsedPercent:%f%%\n", v.Total, v.Free, v.UsedPercent)

		data = "Domos.poolhouse.localTechnique.Raspberry.Memory.Total"
		value = fmt.Sprintf("%v", v.Total)
		err = GraphiteCnx(data, value, graphiteHost, graphitePort)
		if err != nil {
			fmt.Println("Erreur insertion dans graphite")
		}

		data = "Domos.poolhouse.localTechnique.Raspberry.Memory.Free"
		value = fmt.Sprintf("%v", v.Free)
		err = GraphiteCnx(data, value, graphiteHost, graphitePort)
		if err != nil {
			fmt.Println("Erreur insertion dans graphite")
		}

		data = "Domos.poolhouse.localTechnique.Raspberry.Memory.Pourcent"
		value = fmt.Sprintf("%.2f", v.UsedPercent)
		err = GraphiteCnx(data, value, graphiteHost, graphitePort)
		if err != nil {
			fmt.Println("Erreur insertion dans graphite")
		}

		// https://github.com/shirou/gopsutil/blob/master/net/net.go
		nv, _ := netio.IOCounters(true)

		for idx, _ := range nv {
			//fmt.Println(idx)
			vName := fmt.Sprintf("%v", nv[idx].Name)
			BytesRecv := fmt.Sprintf("%v", nv[idx].BytesRecv)
			BytesSent := fmt.Sprintf("%v", nv[idx].BytesSent)

			data = "Domos.poolhouse.localTechnique.Raspberry.NET." + vName + ".BytesRecv"

			//fmt.Printf(" * Network: %s %v bytes / %v bytes\n", nv[idx].Name, nv[idx].BytesRecv, nv[idx].BytesSent)

			err = GraphiteCnx(data, BytesRecv, graphiteHost, graphitePort)
			if err != nil {
				fmt.Println("Erreur insertion dans graphite")
			}

			data = "Domos.poolhouse.localTechnique.Raspberry.NET." + vName + ".BytesSent"

			err = GraphiteCnx(data, BytesSent, graphiteHost, graphitePort)
			if err != nil {
				fmt.Println("Erreur insertion dans graphite")
			}
		}

		time.Sleep(dureeMetrcisInt * time.Millisecond)
	}
}
