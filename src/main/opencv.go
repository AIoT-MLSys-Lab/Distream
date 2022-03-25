package main

import (
	"Distream/src/tools"
	"fmt"
	"gocv.io/x/gocv"
	"image"
	//"image"
	//"image/color"
)

func readVideo(windowName string) {
	// parse args
	deviceID := 0

	//webcam, err := gocv.VideoCaptureDevice(int(deviceID))
	webcam, err := gocv.VideoCaptureFile(fmt.Sprintf("/home/xiao/tmp/traffic%v.mp4", windowName))

	if err != nil {
		fmt.Printf("Error opening video capture device: %v\n", deviceID)
		return
	}
	defer webcam.Close()

	window := gocv.NewWindow(windowName)
	defer window.Close()

	img := gocv.NewMat()
	defer img.Close()

	imgDelta := gocv.NewMat()
	defer imgDelta.Close()

	imgThresh := gocv.NewMat()
	defer imgThresh.Close()

	mog2 := gocv.NewBackgroundSubtractorMOG2()
	defer mog2.Close()

	fmt.Printf("Start reading camera device: %v\n", deviceID)
	for {
		t1 := tools.MakeTimestamp()
		if ok := webcam.Read(&img); !ok {
			fmt.Printf("Error cannot read device %d\n", deviceID)
			return
		}
		if img.Empty() {
			continue
		}

		// first phase of cleaning up image, obtain foreground only
		mog2.Apply(img, &imgDelta)

		fmt.Println("step 1", tools.MakeTimestamp()-t1)
		// remaining cleanup of the image to use for finding contours.
		// first use threshold
		gocv.Threshold(imgDelta, &imgThresh, 25, 255, gocv.ThresholdBinary)

		// then dilate
		kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
		defer kernel.Close()
		gocv.Dilate(imgThresh, &imgThresh, kernel)
		fmt.Println("step 2", tools.MakeTimestamp()-t1)
		//now find contours
		gocv.FindContours(imgThresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)
		fmt.Println("step 3", tools.MakeTimestamp()-t1)
		//fmt.Println(len(contours))
		//for _, c := range contours {
		//	area := gocv.ContourArea(c)
		//	if area < 5000 {
		//		continue
		//	}
		//
		//	//status = "Motion detected"
		//	//statusColor = color.RGBA{255, 0, 0, 0}
		//	//gocv.DrawContours(&img, contours, i, statusColor, 2)
		//	//
		//	//rect := gocv.BoundingRect(c)
		//	//gocv.Rectangle(&img, rect, color.RGBA{0, 0, 255, 0}, 2)
		//}

		//window.IMShow(img)
		a := 66 - int(tools.MakeTimestamp()-t1)
		if a <= 0 {
			a = 1
		}
		fmt.Println(windowName, "time", a)
		//if window.WaitKey(a) == 27 {
		//	break
		//}

	}

}

func main_0() {
	//done := make(chan bool)
	go readVideo("1")
	go readVideo("2")
	go readVideo("3")
	readVideo("4")

}
