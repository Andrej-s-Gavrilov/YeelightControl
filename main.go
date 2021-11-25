package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"

	cfg "yeelight_control/config"
	tcp "yeelight_control/tcp_connection"
)

var (
	application_name    string
	version             string
	mqtt_client         string
	mqtt_client_version string
	mqtt_broker         string
	mqtt_port           int
	config_file_path    string
	log_filename        string
	get_version         bool
	configuration       cfg.Config
	log_level           log.Level
	lamp_pool           tcp.LampPool
	client              mqtt.Client
	osSignals           = make(chan os.Signal, 1)

	// {"id":1,"method":"adjust_bright","params":[20, 500]}
	// {"id":1,"method":"adjust_bright","params":[-20, 500]}
	// {"id":1,"method":"set_power","params":["on", "smooth", 500]}
	// {"id":1,"method":"set_power","params":["off", "smooth", 500]}
	cmd = map[string]string{
		"adj_bp": "{\"id\":1,\"method\":\"adjust_bright\",\"params\":[10, 500]}",
		"adj_bm": "{\"id\":1,\"method\":\"adjust_bright\",\"params\":[-10, 500]}",
		"on":     "{\"id\":1,\"method\":\"set_power\",\"params\":[\"on\", \"smooth\", 2500]}",
		"off":    "{\"id\":1,\"method\":\"set_power\",\"params\":[\"off\", \"smooth\", 500]}",
	}
)

func init() {
	const (
		default_application_name = "YeelightControl-dev"
		default_version          = "0.0-dev"
		default_config_file_path = ".\\config.yml"
		default_log_file_path    = "yeelightControl.log"
		default_log_level        = log.InfoLevel
	)
	rand.Seed(time.Now().UTC().UnixNano())

	// Parse comand line parameters
	flag.StringVar(&config_file_path, "config", default_config_file_path, "Path to configuration file")
	flag.BoolVar(&get_version, "version", false, "Print version of application")
	flag.Parse()

	// Load configuration
	configuration, err := cfg.NewConfig(config_file_path)
	if err != nil {
		fmt.Println("Error during parsing/opening configuration-file. Error: " + err.Error())
		os.Exit(1)
	}
	cfg.ValidateConfig(*configuration)

	// Configure loging
	if configuration.Loging.Log_file_path != "" {
		log_filename = configuration.Loging.Log_file_path
	} else {
		log_filename = default_log_file_path
	}

	switch configuration.Loging.Log_level {
	case "debug":
		log_level = log.InfoLevel
	case "info":
		log_level = log.InfoLevel
	case "error":
		log_level = log.ErrorLevel
	default:
		log_level = default_log_level
	}

	if file, err := os.OpenFile(log_filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err != nil {
		fmt.Println("Error during opening log-file. Error: " + err.Error())
		os.Exit(1)
	} else {
		log.SetOutput(file)
		log.SetLevel(log.InfoLevel)
		//TODO: need to close log-file
	}

	application_name = default_application_name
	version = default_version

	fmt.Printf("[%s] YeelightControl is starting ...\n", time.Now().Format("15:04:05.000"))
	log.WithFields(log.Fields{"modul": "main.init"}).Info(application_name + ". Ver: " + version + "is starting ...")
	log.WithFields(log.Fields{"modul": "main.init"}).Info("Logleve: ", log_level.String())

	// Configure MQTT brocker
	// TODO: need to verify variables in ValidateConfig()
	mqtt_broker = configuration.MQTT_Server.Host
	mqtt_port = configuration.MQTT_Server.Port
	mqtt_client = configuration.MQTT_Server.Client_name
	mqtt_client_version = configuration.MQTT_Server.Client_version

	// Configure devices pool
	lamp_pool = tcp.InitLampPool()
	for _, device := range configuration.Devices {
		tcp.AddLamp(&lamp_pool, device.Device_name, device.Host, device.Port, device.Control_topic, device.Satus_topic)
		tcp.ConnectLamp(lamp_pool.Pool[device.Device_name])
	}
}

func main() {
	defer func() {
		fmt.Printf("[%s] YeelightControl is stoping ...\n", time.Now().Format("15:04:05.000"))
		log.WithFields(log.Fields{"modul": "main"}).Info("YeelightControl is stoping ...")

		client.Disconnect(250)
		fmt.Printf("[%s] MQTT client stoped\n", time.Now().Format("15:04:05.000"))
		log.WithFields(log.Fields{"modul": "main"}).Info("MQTT client (" + mqtt_client + " ver:" + mqtt_client_version + ") stoped")

		for _, lamp := range lamp_pool.Pool {
			tcp.DisconnectLamp(lamp) // TODO: need to check active connection
			fmt.Printf("[%s] Lamp %s[%s:%s] disconnected\n", time.Now().Format("15:04:05.000"), lamp.Lamp_name, lamp.Ip, lamp.Port)
			log.WithFields(log.Fields{"modul": "main"}).Info("Lamp " + lamp.Lamp_name + "[" + lamp.Ip + ":" + lamp.Port + "] disconnected")
		}

		fmt.Printf("[%s] YeelightControl stoped\n", time.Now().Format("15:04:05.000"))
		log.WithFields(log.Fields{"modul": "main"}).Info("YeelightControl stoped")
	}()

	if get_version {
		fmt.Println(application_name + version)
		return
	}

	// Work with MQTT broker
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", mqtt_broker, mqtt_port))
	opts.SetClientID(mqtt_client + "-" + mqtt_client_version)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client = mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("[%s] Can not estaplishe connection to MQTT: %s:%s\n", time.Now().Format("15:04:05.000"), mqtt_broker, strconv.Itoa(mqtt_port))
		log.WithFields(log.Fields{"modul": "main"}).Fatal(token.Error().Error())
	}
	fmt.Printf("[%s] MQTT broker: %s:%s. Client: %s (%s)\n", time.Now().Format("15:04:05.000"), mqtt_broker, strconv.Itoa(mqtt_port), mqtt_client, mqtt_client_version)
	log.WithFields(log.Fields{"modul": "main"}).Info("MQTT broker is connected. Broker:" + mqtt_broker + ":" + strconv.Itoa(mqtt_port) + " Client:" + mqtt_client + " ver:" + mqtt_client_version)

	//Subscription to device control topics
	for _, topic := range lamp_pool.Pool {
		sub(client, topic.Mqtt_topic_ctl)
	}

	//For pushing status toward Broker
	//publish(client, lamp_pool.Pool["yeelight1"].Mqtt_topic_ctl)

	fmt.Printf("[%s] YeelightControl started\n", time.Now().Format("15:04:05.000"))
	log.WithFields(log.Fields{"modul": "main"}).Info("YeelightControl started")

	signal.Notify(osSignals, os.Interrupt)
	signal := <-osSignals
	fmt.Printf("[%s] YeelightControl recived signal: %+v signal\n", time.Now().Format("15:04:05.000"), signal)
	log.WithFields(log.Fields{"modul": "main"}).Info("YeelightControl recived signal: " + signal.String())
}

// Tools
func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

//MQTT
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Printf("[%s] MQTT broker: Connected\n", time.Now().Format("15:04:05.000"))
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("[%s] Connect lost: %v\n", time.Now().Format("15:04:05.000"), err)
	log.WithFields(log.Fields{"modul": "main"}).Fatal("Connect lost: " + err.Error())
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("[%s] Received message: %s from topic: %s\n", time.Now().Format("15:04:05.000"), msg.Payload(), msg.Topic())
	lampKey := tcp.GetLampByCtlTopic(&lamp_pool, msg.Topic())
	if lampKey != "" {
		message_id := strconv.Itoa(randInt(1000, 9999))
		cmd := string(msg.Payload())

		log.WithFields(log.Fields{"modul": "main"}).Info("[" + message_id + "] Sending message : " + cmd + " from topic: " + msg.Topic())

		// Try to send command to the Device
		if res, ok := tcp.SendCommandLamp(lamp_pool.Pool[lampKey], cmd); !ok {
			log.WithFields(log.Fields{"modul": "main"}).Warning("[" + message_id + "] Error sending")

			fmt.Printf("[%s] Error sending message: \"%s\" from topic: %s\n", time.Now().Format("15:04:05.000"), cmd, msg.Topic())

			// Need to add reconnection procedure
			// Curently there is one attempt to reconnect
			log.WithFields(log.Fields{"modul": "main"}).Info("[" + message_id + "] Reconnecting")
			if ok := tcp.ConnectLamp(lamp_pool.Pool[lampKey]); ok {
				if res, ok := tcp.SendCommandLamp(lamp_pool.Pool[lampKey], cmd); !ok {
					log.WithFields(log.Fields{"modul": "main"}).Warning("[" + message_id + "] Resend Message. Error sending. Attempt of resend: " + strconv.Itoa(1))
				} else {
					fmt.Printf("[%s] Result: %s\n", time.Now().Format("15:04:05.000"), res)
					log.WithFields(log.Fields{"modul": "main"}).Info("[" + message_id + "] Resend message. Result : " + res)
				}
			} else {
				log.WithFields(log.Fields{"modul": "main"}).Warning("[" + message_id + "] Failed attempt to reconnect")
			}
		} else {
			fmt.Printf("[%s] Result: %s\n", time.Now().Format("15:04:05.000"), res)
			log.WithFields(log.Fields{"modul": "main"}).Info("[" + message_id + "] Result : " + res)
		}
	} else {
		log.WithFields(log.Fields{"modul": "main"}).Warning("There is subscription on topick" + msg.Topic() + ", but Lamp did not finde by ctl topick")
	}
}

func sub(client mqtt.Client, topic string) {
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	log.WithFields(log.Fields{"modul": "main"}).Info("Subscribed to topic: " + topic)
}

func publish(client mqtt.Client, topic string) {
	text := fmt.Sprintf("{\"id\": %d, \"status\": \"ON\"}", 1)
	token := client.Publish(topic, 0, false, text)
	token.Wait()
	time.Sleep(time.Second)
}
