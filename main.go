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

	tcp "yeelight_control/tcp_connection"
)

var (
	application_name    string
	version             string
	mqtt_client         string
	mqtt_client_version string
	mqtt_broker         string
	mqtt_port           int
	config_file_name    string
	log_filename        string
	get_version         bool
	log_level           log.Level
	lamp_pool           tcp.LampPool
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
		default_config_fileName  = "config.yaml"
		default_log_filename     = "yeelightControl.log"
		default_log_level        = log.InfoLevel //log.ErrorLevel, log.DebugLevel
		deafult_mqtt_broker      = "192.168.0.2"
		default_mqtt_port        = 1883
	)
	rand.Seed(time.Now().UTC().UnixNano())

	flag.StringVar(&config_file_name, "config", default_config_fileName, "Name of configuration file")
	flag.BoolVar(&get_version, "version", false, "Print version of application")

	application_name = default_application_name
	version = default_version

	mqtt_client = application_name
	mqtt_client_version = version
	mqtt_broker = deafult_mqtt_broker
	mqtt_port = default_mqtt_port

	// Configure loging
	log_filename = default_log_filename
	log_level = default_log_level
	file, err := os.OpenFile(log_filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to log to file, using default stderr")
	}
	log.SetLevel(log.InfoLevel)
}

func main() {
	defer func() {
		fmt.Printf("[%s] YeelightControl stoped\n", time.Now().Format("15:04:05.000"))
		log.WithFields(log.Fields{"modul": "main"}).Info("YeelightControl stoped")
	}()

	flag.Parse()

	if get_version {
		fmt.Println("Application version is " + application_name + version)
		os.Exit(0)
	}

	fmt.Printf("[%s] YeelightControl is starting ...\n", time.Now().Format("15:04:05.000"))
	log.WithFields(log.Fields{"modul": "main"}).Info("YeelightControl is starting ...")

	// Work with Lamp
	lamp_pool = tcp.InitLampPool()

	// My configuration
	//tcp.AddLamp(&lamp_pool, "yeelight1", "192.168.0.3", "55443", "yeelightControl/yeelight1/control", "yeelightControl/yeelight1/status")
	//tcp.ConnectLamp(lamp_pool.Pool["yeelight1"])

	// Test configuration
	tcp.AddLamp(&lamp_pool, "yeelightTest1", "127.0.0.1", "55001", "yeelightControl/yeelightTest1/control", "yeelightControl/yeelightTest1/status")
	tcp.AddLamp(&lamp_pool, "yeelightTest2", "127.0.0.1", "55002", "yeelightControl/yeelightTest2/control", "yeelightControl/yeelightTest2/status")
	tcp.AddLamp(&lamp_pool, "yeelightTest3", "127.0.0.1", "55003", "yeelightControl/yeelightTest3/control", "yeelightControl/yeelightTest3/status")
	tcp.ConnectLamp(lamp_pool.Pool["yeelightTest1"])
	tcp.ConnectLamp(lamp_pool.Pool["yeelightTest2"])
	tcp.ConnectLamp(lamp_pool.Pool["yeelightTest3"])

	// Work with MQTT broker
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", mqtt_broker, mqtt_port))
	opts.SetClientID(mqtt_client + "-" + mqtt_client_version)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("[%s] Can not estaplishe connection to MQTT: %s:%s\n", time.Now().Format("15:04:05.000"), mqtt_broker, strconv.Itoa(mqtt_port))
		log.WithFields(log.Fields{"modul": "main"}).Fatal(token.Error().Error())
	}
	fmt.Printf("[%s] MQTT broker: %s:%s. Client: %s (%s)\n", time.Now().Format("15:04:05.000"), mqtt_broker, strconv.Itoa(mqtt_port), mqtt_client, mqtt_client_version)
	log.WithFields(log.Fields{"modul": "main"}).Info("MQTT broker is connected. Broker:" + mqtt_broker + ":" + strconv.Itoa(mqtt_port) + " Client:" + mqtt_client + " ver:" + mqtt_client_version)

	// My configuration - yeelight1
	//sub(client, lamp_pool.Pool["yeelight1"].Mqtt_topic_ctl)

	// Test configuration
	sub(client, lamp_pool.Pool["yeelightTest1"].Mqtt_topic_ctl)
	sub(client, lamp_pool.Pool["yeelightTest2"].Mqtt_topic_ctl)
	sub(client, lamp_pool.Pool["yeelightTest3"].Mqtt_topic_ctl)

	//For pushing status toward Broker
	//publish(client, lamp_pool.Pool["yeelight1"].Mqtt_topic_ctl)

	fmt.Printf("[%s] YeelightControl started\n", time.Now().Format("15:04:05.000"))
	log.WithFields(log.Fields{"modul": "main"}).Info("YeelightControl started")

	signal.Notify(osSignals, os.Interrupt)
	go interruptionHandler(client)
	for {
	}
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func interruptionHandler(client mqtt.Client) {
	signal := <-osSignals
	fmt.Printf("[%s] YeelightControl recived signal: %+v signal\n", time.Now().Format("15:04:05.000"), signal)
	log.WithFields(log.Fields{"modul": "main"}).Info("YeelightControl recived signal: " + signal.String())

	fmt.Printf("[%s] YeelightControl is stoping ...\n", time.Now().Format("15:04:05.000"))
	log.WithFields(log.Fields{"modul": "main"}).Info("YeelightControl is stoping ...")

	client.Disconnect(250)
	fmt.Printf("[%s] MQTT client stoped\n", time.Now().Format("15:04:05.000"))
	log.WithFields(log.Fields{"modul": "main"}).Info("MQTT client (" + mqtt_client + " ver:" + mqtt_client_version + " stoped")

	for _, lamp := range lamp_pool.Pool {
		tcp.DisconnectLamp(lamp)
		fmt.Printf("[%s] Lamp %s[%s:%s] disconnected\n", time.Now().Format("15:04:05.000"), lamp.Lamp_name, lamp.Ip, lamp.Port)
		log.WithFields(log.Fields{"modul": "main"}).Info("Lamp" + lamp.Lamp_name + "[" + lamp.Ip + ":" + lamp.Port + "] disconnected " + mqtt_client + " ver:" + mqtt_client_version + " stoped")
	}
	os.Exit(0)
}

//MQTT
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Printf("[%s] MQTT broker: Connected\n", time.Now().Format("15:04:05.000"))
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("[%s] Connect lost: %v\n", time.Now().Format("15:04:05.000"), err)
	log.WithFields(log.Fields{"modul": "main"}).Fatal("Connect lost: " + err.Error())
}

// TODO: Is never used
func initMQTTconnection(mqtt_broker string, mqtt_port int) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", mqtt_broker, mqtt_port))
	opts.SetClientID(mqtt_client + "-" + mqtt_client_version)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.WithFields(log.Fields{"modul": "main"}).Fatal(token.Error().Error())
		panic(token.Error())
	}
	go interruptionHandler(client)
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("[%s] Received message: %s from topic: %s\n", time.Now().Format("15:04:05.000"), msg.Payload(), msg.Topic())
	lampKey := tcp.GetLampByCtlTopic(&lamp_pool, msg.Topic())
	if lampKey != "" {
		message_id := strconv.Itoa(randInt(1000, 9999))
		cmd := string(msg.Payload())

		log.WithFields(log.Fields{"modul": "main"}).Info("[" + message_id + "] Sending message : " + cmd + " from topic: " + msg.Topic())

		// Try to send command to the Lamp
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
			log.WithFields(log.Fields{"modul": "main"}).Info("[" + message_id + " Result : " + res)
		}
	} else {
		log.WithFields(log.Fields{"modul": "main"}).Warning("There is subscription on topick" + msg.Topic() + ", but Lamp did not finde by ctl topick")
	}
}

func sub(client mqtt.Client, topic string) {
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	//fmt.Printf("[%s] Subscribed to topic %s\n", time.Now().Format("15:04:05.000"), topic)
	log.WithFields(log.Fields{"modul": "main"}).Info("Subscribed to topic: " + topic)
}

func publish(client mqtt.Client, topic string) {
	text := fmt.Sprintf("{\"id\": %d, \"status\": \"ON\"}", 1)
	token := client.Publish(topic, 0, false, text)
	token.Wait()
	time.Sleep(time.Second)
}

// MQTT -
