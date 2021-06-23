package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"
	tcp "yeelight_control/tcp_connection"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

var (
	mqtt_broker = "192.168.0.2"
	mqtt_port   = 1883
	mqtt_ports  = "1883"
	//mqtt_client = "YeelightControl"
	//mqtt_client_version = "0.1"
	mqtt_client         = "YeelightControl-dev"
	mqtt_client_version = "0.0-dev"
	osSignals           = make(chan os.Signal, 1)

	log_filename = "YeelightControl.log"

	lamp_pool tcp.LampPool
	//connPool     *tcp.ConnectionPool

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
	fmt.Printf("\n[%s] Init() YeelightControl started\n", time.Now().Format("15:04:05.000"))
	rand.Seed(time.Now().UTC().UnixNano())
	file, err := os.OpenFile(log_filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to log to file, using default stderr")
	}

	// Lof level
	//log.SetLevel(log.ErrorLevel)
	log.SetLevel(log.InfoLevel)
	//log.SetLevel(log.DebugLevel)
}

func main() {
	defer func() {
		fmt.Printf("[%s] YeelightControl stoped\n", time.Now().Format("15:04:05.000"))
		log.WithFields(log.Fields{"modul": "main"}).Info("YeelightControl stoped")
	}()
	log.WithFields(log.Fields{"modul": "main"}).Info("YeelightControl started")

	// Work with Lamp
	lamp_pool = tcp.InitLampPool()
	// Lamp-1
	//tcp.AddLamp(&lamp_pool, "yeelight1", "192.168.0.3", "55443", "yeelightControl/yeelight1/control", "yeelightControl/yeelight1/status")
	//tcp.ConnectLamp(lamp_pool.Pool["yeelight1"])

	// Test config
	tcp.AddLamp(&lamp_pool, "yeelightTest1", "127.0.0.1", "55001", "yeelightControl/yeelightTest1/control", "yeelightControl/yeelightTest1/status")
	tcp.AddLamp(&lamp_pool, "yeelightTest2", "127.0.0.1", "55002", "yeelightControl/yeelightTest2/control", "yeelightControl/yeelightTest2/status")
	tcp.AddLamp(&lamp_pool, "yeelightTest3", "127.0.0.1", "55003", "yeelightControl/yeelightTest3/control", "yeelightControl/yeelightTest3/status")
	tcp.ConnectLamp(lamp_pool.Pool["yeelightTest1"])
	tcp.ConnectLamp(lamp_pool.Pool["yeelightTest2"])
	tcp.ConnectLamp(lamp_pool.Pool["yeelightTest3"])

	// MQTT broker

	fmt.Printf("[%s] MQTT broker: %s:%s. Client: %s (%s)\n", time.Now().Format("15:04:05.000"), mqtt_broker, mqtt_ports, mqtt_client, mqtt_client_version)
	log.WithFields(log.Fields{"modul": "main"}).Info("MQTT broker: " + mqtt_broker + ":" + mqtt_ports + " Client:" + mqtt_client + "v" + mqtt_client_version)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", mqtt_broker, mqtt_port))
	opts.SetClientID(mqtt_client + "-" + mqtt_client_version)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("[%s] Can not estaplishe connection to MQTT: %s:%s\n", time.Now().Format("15:04:05.000"), mqtt_broker, mqtt_ports)
		log.WithFields(log.Fields{"modul": "main"}).Fatal(token.Error().Error())
		//panic(token.Error())
	}
	go interruptionHandler(client)

	// Work - yeelight1
	//sub(client, lamp_pool.Pool["yeelight1"].Mqtt_topic_ctl)

	//
	sub(client, lamp_pool.Pool["yeelightTest1"].Mqtt_topic_ctl)
	sub(client, lamp_pool.Pool["yeelightTest2"].Mqtt_topic_ctl)
	sub(client, lamp_pool.Pool["yeelightTest3"].Mqtt_topic_ctl)

	//publish(client, lamp_pool.Pool["yeelight1"].Mqtt_topic_ctl)

	/*
		var res string
		var ok bool
		var itr = 0
	*/
	for {
		/* 		itr++
		   		log.WithFields(log.Fields{"modul": "main"}).Debug("Test iteratipon: " + string(itr) + " is going on")
		   		fmt.Printf("[%s] Log DEBUG\n", time.Now().Format("15:04:05.000"))

		   		if res, ok = tcp.SendCommandLamp(lamp_pool.Pool["yeelight1"], "Lamp1 ... I'm here"); !ok {
		   			fmt.Printf("[%s] Error during sending message to Lammp1\n", time.Now().Format("15:04:05.000"))
		   			os.Exit(1)
		   		} else {
		   			fmt.Printf("[%s] Message is sent, lamp: yeelight1. Iteration: %d\n", time.Now().Format("15:04:05.000"), itr)
		   			_ = res
		   		}
		   		time.Sleep(10 * time.Second)

		   		if res, ok = tcp.SendCommandLamp(lamp_pool.Pool["yeelight2"], "Lamp2 ... I'm too"); !ok {
		   			fmt.Printf("[%s] Error during sending message to Lammp2", time.Now().Format("15:04:05.000"))
		   			os.Exit(1)
		   		} else {
		   			fmt.Printf("[%s] Message is sent, lamp: yeelight2. Iteration: %d\n", time.Now().Format("15:04:05.000"), itr)
		   			_ = res
		   		}
		   		time.Sleep(5 * time.Second)
		*/
	}

}

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

func interruptionHandler(client mqtt.Client) {
	signal := <-osSignals
	fmt.Printf("STOPPING %s version:%s (received %+v signal)", mqtt_client, mqtt_client_version, signal)
	client.Disconnect(250)
	log.WithFields(log.Fields{"modul": "main"}).Fatal("STOPPING: " + mqtt_client + " version:" + mqtt_client_version + "(received " + signal.String() + ")")
	//os.Exit(0)
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
	// ????
	//key := GetLampByCtlTopic(lamp_pool, msg.Topic())
	go func() {
		var res string
		var ok bool
		message_id := strconv.Itoa(randInt(1000, 9999))
		cmd := string(msg.Payload())
		for i := 1; i <= 3; i++ {
			log.WithFields(log.Fields{"modul": "main"}).Info("[" + message_id + "]Sending message : " + cmd + " from topic: " + msg.Topic())
			log.WithFields(log.Fields{"modul": "main"}).Info("[" + message_id + "]Attempt to send : " + strconv.Itoa(i))
			//fmt.Printf("[%s] Attempt to send cmd - %d\n", time.Now().Format("15:04:05.000"), i)

			// Todo: Need to use Multi Variable
			if res, ok = tcp.SendCommandLamp(lamp_pool.Pool["yeelight1"], cmd); !ok {
				log.WithFields(log.Fields{"modul": "main"}).Warning("[" + message_id + "]Error sending Attempt of resend: " + strconv.Itoa(i))
				//fmt.Printf("[%s] Error sending\n", time.Now().Format("15:04:05.000"))
				//fmt.Printf("[%s] Trying to reconnect - %d\n", time.Now().Format("15:04:05.000"), i)
				//Need to block other goroutines. We have to avoid multiply reconnection.
				if ok := tcp.ConnectLamp(lamp_pool.Pool["yeelight1"]); !ok {
					time.Sleep(1 * time.Second)
				}
			} else {
				//fmt.Printf("[%s] Result: %s\n", time.Now().Format("15:04:05.000"), res)
				log.WithFields(log.Fields{"modul": "main"}).Info("[" + message_id + "]Result : " + res)
				_ = res
				break
			}

		}
	}()
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
