package adb_test

import (
	"log"

	adb "github.com/yosemite-open/go-adb"
)

func ExampleDevice_DoSyncLocalFile() {
	adbc, _ := adb.New()
	dev := adbc.Device(adb.AnyUsbDevice())

	awr, err := dev.DoSyncLocalFile("/data/local/tmp/tmp.txt", "adb.go", 0644)
	if err != nil {
		log.Fatal(err)
	}

Loop:
	for {
		select {
		case <-awr.C:
			log.Printf("transfered %v / %v bytes (%.2f%%)",
				awr.BytesCompleted(),
				awr.TotalSize,
				100*awr.Progress())
		case <-awr.DoneCopy:
			log.Printf("finish io copy")
		case <-awr.Done:
			log.Printf("finish system copy, this is final")
			break Loop
		}
	}
	log.Printf("copy error:", awr.Err())
}
