package v2

import (
	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"gopkg.in/yaml.v2"
)

var catalog map[string]string

func init() {
	catalog = map[string]string{
		"53f9f2c3-8b27-4f93-981c-8eac2639600a": `
        # yaml-language-server: $schema=https://raw.githubusercontent.com/tigerinus/CasaOS-AppStore-Hub/init/schemas/compose-x-casaos-spec.json
        services:
          app:
            image: linuxserver/syncthing:latest
            privileged: false
            network_mode: bridge
            environment:
              - TZ=$TZ
              - PUID=$PUID
              - PGID=$PGID
            ports:
              - 8384:8384/tcp
              - 22000:22000/tcp
              - 22000:22000/udp
              - 21027:21027/udp
            volumes:
              - /DATA/AppData/Syncthing/config:/config:rw
              - /DATA:/DATA:rw
            mem_reservation: 256m
            restart: unless-stopped
        
            x-casaos:
              title:
                - lang: en_US
                  text: Syncthing
              name: syncthing
              icon: "https://cdn.jsdelivr.net/gh/IceWhaleTech/CasaOS-AppStore@main/Apps/Syncthing/icon.png"
              tagline:
                - lang: en_US
                  text: Free, secure, and distributed file synchronisation tool.
              overview:
                - lang: en_US
                  text: Syncthing is a continuous file synchronization program. It synchronizes files between two or more computers in real time, safely protected from prying eyes. Your data is your data alone and you deserve to choose where it is stored, whether it is shared with some third party, and how it's transmitted over the internet.
              thumbnail: https://cdn.jsdelivr.net/gh/IceWhaleTech/CasaOS-AppStore@main/Apps/Jellyfin/thumbnail.jpg
              screenshots: []
              category:
                - Backup
                - File Sync
              developer:
                name: Syncthing
                website: https://syncthing.net/
              adaptor:
                name: CasaOS Team
                website: https://casaos.io
              support: https://discord.gg/knqAbbBbeX
              website: https://casaos.io
              container:
                shell: bash
                web_ui:
                  http: 8384
                  path: /
                envs:
                  - key: TZ
                    configurable: false
                    description:
                      - lang: en_US
                        text: Timezone
                  - key: PUID
                    configurable: false
                    description:
                      - lang: en_US
                        text: Run Syncthing as specified uid.
                  - key: PGID
                    configurable: false
                    description:
                      - lang: en_US
                        text: Run Syncthing as specified gid.
                ports:
                  - container: 8384
                    type: tcp
                    allocation: preferred
                    configurable: advanced
                    description:
                      - lang: en_US
                        text: WebUI HTTP Port
                  - container: 22000
                    type: tcp
                    allocation: required
                    configurable: no
                    description:
                      - lang: en_US
                        text: Syncthing listening port (TCP)
                  - container: 22000
                    type: udp
                    allocation: required
                    configurable: no
                    description:
                      - lang: en_US
                        text: Syncthing listening port (UDP)
                  - container: 21027
                    type: udp
                    allocation: optional
                    configurable: no
                    description:
                      - lang: en_US
                        text: Syncthing protocol discovery port
                volumes:
                  - container: /config
                    allocation: automatic
                    configurable: no
                    description:
                      - lang: en_US
                        text: Syncthing configuration directory
                  - container: /DATA
                    allocation: automatic
                    configurable: advanced
                    description:
                      - lang: en_US
                        text: Syncthing accessible directory
                constraints:
                  min_storage: 1024m        
        `,
	}
}

func GetAppInfo(id codegen.StoreAppID) error {
	composeYAML := GetAppComposeYAML(id)

	var compose interface{}

	if err := yaml.Unmarshal([]byte(*composeYAML), &compose); err != nil {
		return err
	}

	return nil
}

func GetAppComposeYAML(id codegen.StoreAppID) *string {
	if v, ok := catalog[id]; ok {
		return &v
	}

	return nil
}
