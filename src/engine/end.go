package engine

import (
	"Distream/src/rpc"
	"Distream/src/tools"
	"encoding/gob"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var MAX_END_QUEUE = 100

type End struct {
	id                int
	client            *rpc.RPCClient
	InQueue           chan Query
	netQueue          chan Query
	BaseParPoint      float64
	FineGrainedOffset float64
	SubmitQuery       func(Query) error
	Capacity          float64 //0 - 100
	//Workload    float64
	VideoFolder       string
	frameMap          map[int]map[int]int
	numProcessed      int
	dataset           string
	processTimeMovAvg *tools.MovingAvg
	model             *Model
}

func NewEnd(id int, dataset string, serverAddr string) *End {

	// init RPC client
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		panic(err)
	}
	client := rpc.NewClient(id, conn)

	var videoFolder string
	switch dataset {
	case "campus":
		videoFolder = fmt.Sprintf("/home/xiao/projects/DeepEdge/dataset/campus_A/CAM%02d", id+1)
	case "traffic":
		videoFolder = fmt.Sprintf("/home/xiao/projects/DeepEdge/dataset/traffic_A/CAM%02d", id+1)
	default:
		err := fmt.Errorf("Unknown dataset: %v", dataset)
		panic(err)
	}

	end := End{id: id, client: client, InQueue: make(chan Query, MAX_END_QUEUE),
		netQueue: make(chan Query, 100), Capacity: 100, dataset: dataset,
		VideoFolder:       videoFolder,
		processTimeMovAvg: &tools.MovingAvg{WindowLenghth: 100, UpdateChan: make(chan float64)},
		model:             NewModel()}

	end.prepareVideoFrames()

	//register remote services to local field
	end.client.CallRPC("SubmitQuery", &end.SubmitQuery)

	return &end
}

func (end *End) prepareVideoFrames() {
	var files []string
	end.frameMap = make(map[int]map[int]int)
	err := filepath.Walk(end.VideoFolder, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	})
	if err != nil {
		panic(err)
	}

	persistFileName := fmt.Sprintf("data/%s_%d", end.dataset, end.id+1)
	if _, err := os.Stat(persistFileName); os.IsNotExist(err) {
		for _, file := range files {
			file = filepath.Base(file)
			if filepath.Ext(file) != ".jpg" && filepath.Ext(file) != ".JPG" {
				continue
			}
			tmp := strings.Split(file, "_")
			frame_id, _ := strconv.Atoi(tmp[1])
			if end.frameMap[frame_id] == nil {
				end.frameMap[frame_id] = make(map[int]int)
			}
			class_id, _ := strconv.Atoi(tmp[3])
			end.frameMap[frame_id][class_id] += 1
		}

		dataFile, err := os.Create(persistFileName)
		defer dataFile.Close()
		if err != nil {
			panic(err)
		}
		dataEncoder := gob.NewEncoder(dataFile)
		dataEncoder.Encode(end.frameMap)
	} else {
		dataFile, err := os.Open(persistFileName)
		defer dataFile.Close()
		if err != nil {
			panic(err)
		}
		dataDecoder := gob.NewDecoder(dataFile)
		dataDecoder.Decode(&end.frameMap)
	}

}

func (end *End) InferLoop() {
	for {
		// retrieve query from queue
		select {
		case query := <-end.InQueue:
			//process query
			t1 := tools.MakeTimestamp()
			end.process(&query)
			t2 := tools.MakeTimestamp()

			end.processTimeMovAvg.UpdateChan <- (t2 - t1)
			end.numProcessed += 1
			end.netQueue <- query
		}
	}
}

func (end *End) TransferLoop() error {
	for {
		q := <-end.netQueue
		err := end.SubmitQuery(q)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
	}
}

func (end *End) ReadVideoLoop(maxFrame int) {

	var FrameSkipPercentage = 1.0
	if end.dataset == "campus" {
		FrameSkipPercentage = 40.0 // campus
	} else if end.dataset == "traffic" {
		FrameSkipPercentage = 30.0 // traffic
	}

	for i := 0; i < maxFrame; i++ {
		timer := time.After(1000 / 25 * time.Millisecond)
		if i%(int(100.0/FrameSkipPercentage)) == 0 {
			// do nothing
		} else if objMap, ok := end.frameMap[i]; ok {
			frameQueries := []Query{}
			for k, v := range objMap {
				for i := 0; i < v; i++ {
					switch k {
					case 7:
						frameQueries = append(frameQueries, *NewQuery("person", end.id))
						frameQueries = append(frameQueries, *NewQuery("person", end.id))
					case 0:
						frameQueries = append(frameQueries, *NewQuery("other", end.id))
					default:
						frameQueries = append(frameQueries, *NewQuery("car", end.id))
						frameQueries = append(frameQueries, *NewQuery("car", end.id))
					}
				}
			}
			rand.Shuffle(len(frameQueries), func(i, j int) {
				frameQueries[i],
					frameQueries[j] = frameQueries[j], frameQueries[i]
			})
			go func() {
				for _, query := range frameQueries {
					end.InQueue <- query
				}
			}()
		}
		<-timer
	}
}

func (end *End) getParPoint() (parPoint float64) {
	parPoint = end.BaseParPoint + end.FineGrainedOffset
	if parPoint < 0 {
		parPoint = 0
	} else if parPoint > 1 {
		parPoint = 1
	}
	return parPoint
}

func (end *End) process(query *Query) {

	end.processQuery(query)
}

func (end *End) getMaxTP() float64 {
	if end.numProcessed == 0 {
		return 0
	}
	if end.processTimeMovAvg.Value() == 0.0 {
		return 60000.0
	} else {
		return 1000.0 / end.processTimeMovAvg.Value()
	}
}

func (end *End) RunInBackground() {
	go end.InferLoop()
	go end.TransferLoop()
	go end.ReadVideoLoop(25 * 60 * 1000)
	go end.processTimeMovAvg.UpdateMovAvgLoop(500)
}
