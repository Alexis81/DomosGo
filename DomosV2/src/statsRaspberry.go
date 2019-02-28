package main

//---------------------------------------------------------------------------------------------------------------------------------------------------------
// Package   : statsRaspberry.go
// Port rest : 1001
// Gets      :
//              /Stats
//---------------------------------------------------------------------------------------------------------------------------------------------------------

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	//"reflect"
	//fmt.Println(reflect.TypeOf(stats))
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"gopkg.in/yaml.v2"
)

var (
	version     = "1.0.1"
	lectureInit LectureInit
)

// Structure du Json
type Stats struct {
	Type                                                            string `json:"type"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Cpu              string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Cpu"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Total        string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Total"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Available    string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Available"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Used         string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Used"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_UsedPourcent string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.UsedPourcent"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_UsedFree     string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.UsedFree"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_UsedActive   string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.UsedActive"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_UsedInactive string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.UsedInactive"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Wired        string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Wired"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Buffers      string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Buffers"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Cached       string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Cached"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Writeback    string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Writeback"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Dirty        string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Dirty"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_WritebackTmp string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.WritebackTmp"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Shared       string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Shared"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Slab         string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.Slab"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_PageTables   string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.PageTables"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_SwapCached   string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Mem.SwapCached"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Temperature      string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Temperature"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Frequence        string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Frequence"`
	Domos_poolhouse_localTechnique_Raspberry_Stats_Elapsed          string `json:"Domos.poolhouse.localTechnique.Raspberry.Stats.Elapsed"`
	Version                                                         string `json:"version"`
}

// Structure pour le fichier de configuration
type LectureInit struct {
	Raspberry struct {
		Url  string `yaml:"url"`
		Port string `yaml:"port"`
	} `yaml:"Raspberry"`
}

// Fonction main
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

	http.HandleFunc(lectureInit.Raspberry.Url, func(w http.ResponseWriter, r *http.Request) {
		api(w, r)
	})

	fmt.Println("Server Stats listen port : " + lectureInit.Raspberry.Port)
	http.ListenAndServe(":"+lectureInit.Raspberry.Port, nil)
}

// Fonction de réponse HTTP
func api(w http.ResponseWriter, r *http.Request) {

	// Démarre chrono
	start := time.Now().UTC()

	// Demande les stats
	percent, _ := cpu.Percent(time.Second, false)
	memory, _ := mem.VirtualMemory()

	// Calcul le temps pour lecture température
	elapse := time.Since(start)
	elapseSeconds := fmt.Sprintf("%.3f", elapse.Seconds())

	// Température CPU
	re := regexp.MustCompile("\\d+(\\.\\d+)?")
	// 50464
	out, err := exec.Command("cat", "/sys/class/thermal/thermal_zone0/temp").Output()
	if err != nil {
		fmt.Println(err)
	}
	temperature, err := strconv.ParseFloat(re.FindString(string(out)), 64)
	if err != nil {
		fmt.Println("cmd err")
	}

	temperatureDegrees := strconv.FormatFloat(temperature/1000, 'f', 2, 64)

	// Fréquence CPU
	out, err = exec.Command("cat", "/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_cur_freq").Output()
	if err != nil {
		fmt.Println(err)
	}
	frequence, err := strconv.ParseFloat(re.FindString(string(out)), 64)
	if err != nil {
		fmt.Println("cmd err")
	}

	// Création du JSON
	donnees := Stats{
		Type: "Stats",
		Domos_poolhouse_localTechnique_Raspberry_Stats_Cpu:              fmt.Sprintf("%.3f", percent[0]),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Total:        fmt.Sprintf("%v", memory.Total),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Available:    fmt.Sprintf("%v", memory.Available),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Used:         fmt.Sprintf("%v", memory.Used),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_UsedPourcent: fmt.Sprintf("%v", memory.UsedPercent),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_UsedFree:     fmt.Sprintf("%v", memory.Free),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_UsedActive:   fmt.Sprintf("%v", memory.Active),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_UsedInactive: fmt.Sprintf("%v", memory.Inactive),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Wired:        fmt.Sprintf("%v", memory.Wired),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Buffers:      fmt.Sprintf("%v", memory.Buffers),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Cached:       fmt.Sprintf("%v", memory.Cached),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Writeback:    fmt.Sprintf("%v", memory.Writeback),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Dirty:        fmt.Sprintf("%v", memory.Dirty),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_WritebackTmp: fmt.Sprintf("%v", memory.WritebackTmp),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Shared:       fmt.Sprintf("%v", memory.Shared),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_Slab:         fmt.Sprintf("%v", memory.Slab),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_PageTables:   fmt.Sprintf("%v", memory.PageTables),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Mem_SwapCached:   fmt.Sprintf("%v", memory.SwapCached),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Temperature:      temperatureDegrees,
		Domos_poolhouse_localTechnique_Raspberry_Stats_Frequence:        fmt.Sprintf("%.3f", frequence),
		Domos_poolhouse_localTechnique_Raspberry_Stats_Elapsed:          elapseSeconds,
		Version: version,
	}

	jsonData, _ := json.Marshal(donnees)

	w.Header().Set("content-type", "application/json")
	w.Write([]byte(jsonData))

}
