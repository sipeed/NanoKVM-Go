package application

import (
	"encoding/json"

	"NanoKVM-Server/service/ws"

	log "github.com/sirupsen/logrus"
)

func broadcastApplicationUpdateStatus(status ApplicationUpdateStatus) {
	data, err := json.Marshal(status)
	if err != nil {
		return
	}
	clients := ws.GetManager().GetClients()
	for _, client := range clients {
		if err := client.Write("update", string(data)); err != nil {
			log.Errorf("send application update status to client: %s", err.Error())
		}
	}
}
