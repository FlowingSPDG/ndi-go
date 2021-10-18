/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"crypto/rand"
	"log"
	"os"
	"path"

	"github.com/FlowingSPDG/ndi-go"
)

const ndiLibName = "Processing.NDI.Lib.x64.dll"

func initializeNDI() {
	libDir := os.Getenv("NDI_RUNTIME_DIR_V5")
	if libDir == "" {
		log.Fatalln("ndi sdk is not installed")
	}

	if err := ndi.LoadAndInitialize(path.Join(libDir, ndiLibName)); err != nil {
		log.Fatalln(err)
	}
}

func main() {
	initializeNDI()

	pool := ndi.NewObjectPool()
	settings := pool.NewSendCreateSettings("ndi-go test", "", true, false)
	inst := ndi.NewSendInstance(settings)
	if inst == nil {
		log.Fatalln("could not create sender")
	}

	frame := ndi.NewVideoFrameV2()
	frame.FourCC = ndi.FourCCTypeBGRX
	frame.FrameFormatType = ndi.FrameFormatInterleaved
	frame.Xres = 720
	frame.Yres = 480
	frame.LineStride = frame.Xres * 4

	frameData := make([]byte, frame.Xres*frame.Yres*4)
	frame.Data = &frameData[0]

	defer func() {
		inst.Destroy()
		ndi.DestroyAndUnload()
	}()

	log.Println("Streaming video...")

	for {
		if _, err := rand.Read(frameData); err != nil {
			log.Fatalln(err)
		}

		inst.SendVideoV2(frame)
	}
}
