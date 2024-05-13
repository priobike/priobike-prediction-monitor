package status

import (
	"monitor/log"
	"time"
)

// Continuously log out interesting things.
func Monitor() {
	log.Info.Println("Starting monitor...")
	// Wait a bit initially to let the sync service do its job.
	time.Sleep(20 * time.Second)
	for {
		log.Info.Println("Running monitor...")

		WriteSummary()
		WriteGeoJSONMap()
		WriteStatusForEachSG()

		log.Info.Println("Done running monitor.")
		// Sleep for 1 minute.
		time.Sleep(1 * time.Minute)
	}
}
