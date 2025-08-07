package controllers

import (
	"PrometheusAlert/models"
	"PrometheusAlert/models/elastic"
	"bytes"
	"encoding/json"
	"fmt"
	tmplhtml "html/template"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

type PrometheusAlertController struct {
	beego.Controller
}

/*
准备新增阿里云告警回调
Content-Type: application/x-www-form-urlencoded; charset=UTF-8

expression=$Average>=95
&metricName=Host.mem.usedutilization
&instanceName=instance-name-****
&signature=eEq1zHuCUp0XSmLD8p8VtTKF****
&metricProject=acs_ecs
&userId=12****
&curValue=97.39
&alertName=基础监控-ECS-内存使用率
&namespace=acs_ecs
&triggerLevel=WARN
&alertState=ALERT
&preTriggerLevel=WARN
&ruleId=applyTemplateee147e59-664f-4033-a1be-e9595746****
&dimensions={userId=12****), instanceId=i-12****}
&timestamp=1508136760
*/
type AliyunAlert struct {
	Expression      string `json:"expression"`
	MetricName      string `json:"metricName"`
	InstanceName    string `json:"instanceName"`
	Signature       string `json:"signature"`
	MetricProject   string `json:"metricProject"`
	UserId          string `json:"userId"`
	CurValue        string `json:"curValue"`
	AlertName       string `json:"alertName"`
	Namespace       string `json:"namespace"`
	TriggerLevel    string `json:"triggerLevel"`
	AlertState      string `json:"alertState"`
	PreTriggerLevel string `json:"preTriggerLevel"`
	RuleId          string `json:"ruleId"`
	Dimensions      string `json:"dimensions"`
	Timestamp       string `json:"timestamp"`
}

type PrometheusAlertMsg struct {
	Tpl                string
	Type               string
	Ddurl              string
	Wxurl              string
	Fsurl              string
	Phone              string
	WebHookUrl         string
	ToUser             string
	Email              string
	EmailTitle         string
	ToParty            string
	ToTag              string
	GroupId            string
	AtSomeOne          string
	RoundRobin         string
	Split              string
	WebhookContentType string
}

func (c *PrometheusAlertController) PrometheusAlert() {
	logsign := "[" + LogsSign() + "]"
	var p_json interface{}
	//针对prometheus的消息特殊处理
	p_alertmanager_json := make(map[string]interface{})
	pMsg := PrometheusAlertMsg{}
	logs.Debug(logsign, strings.Replace(string(c.Ctx.Input.RequestBody), "\n", "", -1))
	if c.Input().Get("from") == "aliyun" {
		models.AlertsFromCounter.WithLabelValues("aliyun").Add(1)
		ChartsJson.Aliyun += 1
		//阿里云云监控告警消息处理
		AliyunAlertJson := AliyunAlert{}
		AliyunAlertJson.Expression = c.Input().Get("expression")
		AliyunAlertJson.MetricName = c.Input().Get("metricName")
		AliyunAlertJson.InstanceName = c.Input().Get("instanceName")
		AliyunAlertJson.Signature = c.Input().Get("signature")
		AliyunAlertJson.MetricProject = c.Input().Get("metricProject")
		AliyunAlertJson.UserId = c.Input().Get("userId")
		AliyunAlertJson.CurValue = c.Input().Get("curValue")
		AliyunAlertJson.AlertName = c.Input().Get("alertName")
		AliyunAlertJson.Namespace = c.Input().Get("namespace")
		AliyunAlertJson.TriggerLevel = c.Input().Get("triggerLevel")
		AliyunAlertJson.AlertState = c.Input().Get("alertState")
		AliyunAlertJson.PreTriggerLevel = c.Input().Get("preTriggerLevel")
		AliyunAlertJson.RuleId = c.Input().Get("ruleId")
		AliyunAlertJson.Dimensions = c.Input().Get("dimensions")
		AliyunAlertJson.Timestamp = c.Input().Get("timestamp")
		p_json = AliyunAlertJson
	} else {
		json.Unmarshal(c.Ctx.Input.RequestBody, &p_json)
		//针对prometheus的消息特殊处理
		json.Unmarshal(c.Ctx.Input.RequestBody, &p_alertmanager_json)
	}

	// alertgroup
	alertgroup := c.Input().Get("alertgroup")
	openAg := beego.AppConfig.String("open-alertgroup")
	var agMap map[string]string
	if openAg == "1" && len(alertgroup) != 0 {
		agMap = Alertgroup(alertgroup)
	}

	pMsg.Type = c.Input().Get("type")
	pMsg.Tpl = c.Input().Get("tpl")

	// 告警组适合处理以逗号分隔的多个值
	pMsg.Ddurl = checkURL(agMap["ddurl"], c.Input().Get("ddurl"), beego.AppConfig.String("ddurl"))
	pMsg.Wxurl = checkURL(agMap["wxurl"], c.Input().Get("wxurl"), beego.AppConfig.String("wxurl"))
	pMsg.Fsurl = checkURL(agMap["fsurl"], c.Input().Get("fsurl"), beego.AppConfig.String("fsurl"))
	pMsg.Email = checkURL(agMap["email"], c.Input().Get("email"), beego.AppConfig.String("email"))
	pMsg.GroupId = checkURL(agMap["groupid"], c.Input().Get("groupid"), beego.AppConfig.String("BDRL_ID"))

	pMsg.Phone = checkURL(agMap["phone"], c.Input().Get("phone"))
	if pMsg.Phone == "" && (pMsg.Type == "txdx" || pMsg.Type == "hwdx" || pMsg.Type == "bddx" || pMsg.Type == "alydx" || pMsg.Type == "txdh" || pMsg.Type == "alydh" || pMsg.Type == "rlydh" || pMsg.Type == "7moordx" || pMsg.Type == "7moordh") {
		pMsg.Phone = GetUserPhone(1)
	}

	pMsg.WebHookUrl = checkURL(agMap["webhookurl"], c.Input().Get("webhookurl"))
	// webhookContenType, rr, split, workwechat 是单个值，因此不写入告警组。
	pMsg.WebhookContentType = c.Input().Get("webhookContentType")

	pMsg.ToUser = checkURL(c.Input().Get("wxuser"), beego.AppConfig.String("WorkWechat_ToUser"))
	pMsg.ToParty = checkURL(c.Input().Get("wxparty"), beego.AppConfig.String("WorkWechat_ToUser"))
	pMsg.ToTag = checkURL(c.Input().Get("wxtag"), beego.AppConfig.String("WorkWechat_ToUser"))
	pMsg.EmailTitle = checkURL(c.Input().Get("emailtitle"), beego.AppConfig.String("Email_title"))

	// dd, wx, fsv2 的 at 格式不一样，放在告警组里不好处理和组装。
	pMsg.AtSomeOne = c.Input().Get("at")
	pMsg.RoundRobin = c.Input().Get("rr")
	//该配置仅适用于alertmanager的消息,用于判断是否需要拆分alertmanager告警消息
	pMsg.Split = c.Input().Get("split")

	//模版加载进内存处理,防止告警过多频繁查库
	var PrometheusAlertTpl *models.PrometheusAlertDB
	if GlobalPrometheusAlertTpl == nil {
		GlobalPrometheusAlertTpl, _ = models.GetAllTpl()
	}
	for _, Tpl := range GlobalPrometheusAlertTpl {
		if Tpl.Tplname == pMsg.Tpl {
			PrometheusAlertTpl = Tpl
		}
	}

	var message string
	if pMsg.Type != "" && PrometheusAlertTpl != nil {
		//判断是否是来自 Prometheus的告警
		if pMsg.Split != "false" && PrometheusAlertTpl.Tpluse == "Prometheus" {
			//判断告警路由AlertRouter列表是否为空
			if GlobalAlertRouter == nil {
				query := models.AlertRouterQuery{}
				query.Name = c.GetString("name", "")
				query.Webhook = c.GetString("webhook", "")
				//刷新告警路由AlertRouter
				GlobalAlertRouter, _ = models.GetAllAlertRouter(query)
			}
			Alerts_Value, _ := p_alertmanager_json["alerts"].([]interface{})
			//拆分告警消息
				for _, AlertValue := range Alerts_Value {
					// 创建新的告警 JSON，确保每次处理一条告警
					alertJson := make(map[string]interface{})
					for k, v := range p_alertmanager_json {
						if k != "alerts" {
							alertJson[k] = v
						}
					}
					alertJson["alerts"] = []interface{}{AlertValue} // 只包含当前告警
					go SetRecord(AlertValue)
					// 提取 prometheus 告警消息中的 label，用于和告警路由比对
					xalert := AlertValue.(map[string]interface{})
					// 路由处理，可能存在多个路由都匹配成功，所以这里返回的是个列表 sMsg
					Return_pMsgs := AlertRouterSet(xalert, pMsg, PrometheusAlertTpl.Tpl)
					for _, Return_pMsg := range Return_pMsgs {
						// 获取渲染后的模版
						err, msg := TransformAlertMessage(alertJson, Return_pMsg.Tpl)
						if err != nil {
							// 失败不发送消息
							logs.Error(logsign, err.Error())
							message = err.Error()
						} else {
							// 发送消息
							message = SendMessagePrometheusAlert(msg, &Return_pMsg, logsign)
						}
					}
				}
		} else {
			//获取渲染后的模版
			err, msg := TransformAlertMessage(p_json, PrometheusAlertTpl.Tpl)

			if err != nil {
				logs.Error(logsign, err.Error())
				message = err.Error()
			} else {
				//发送消息
				message = SendMessagePrometheusAlert(msg, &pMsg, logsign)
			}
		}

	} else {
		message = "自定义模板接口参数异常！"
		logs.Error(logsign, message)
	}
	c.Data["json"] = message
	c.ServeJSON()
}

// 路由处理
func AlertRouterSet(xalert map[string]interface{}, PMsg PrometheusAlertMsg, Tpl string) []PrometheusAlertMsg {
	return_Msgs := []PrometheusAlertMsg{}
	is_matched := false
	//循环检测现有的路由规则，找到匹配的目标后，替换发送目标参数
	for _, router_value := range GlobalAlertRouter {
		LabelMap := []LabelMap{}
		//将rules转换为列表
		json.Unmarshal([]byte(router_value.Rules), &LabelMap)
		rules_num := len(LabelMap)
		rules_num_match := 0

		//判断如果是恢复告警, 并且设置不发送恢复告警, 则跳过
		if xalert["status"] == "resolved" && router_value.SendResolved == false {
			alertName := xalert["labels"].(map[string]interface{})["alertname"].(string)
			logs.Info("告警名称：", alertName, "路由规则：", router_value.Name, "路由类型：", router_value.Tpl.Tpltype, "路由恢复告警：", router_value.SendResolved)
			continue
		}

        // 记录告警的 labels，用于日志对比
        alertLabels, _ := json.Marshal(xalert["labels"])
        logs.Debug("处理告警路由【%s】，告警标签：%s, 路由规则：%s", router_value.Name, string(alertLabels), router_value.Rules)

		for _, rule := range LabelMap {
            labelFound := false
            for label_key, label_value := range xalert["labels"].(map[string]interface{}) {
                // 这里需要分两部分处理，一部分是正则规则，一部分是非正则规则
                if rule.Name == label_key {
                    labelFound = true
                    if rule.Regex {
                        // 正则部分比对
                        tz := regexp.MustCompile(rule.Value)
                        if len(tz.FindAllString(label_value.(string), -1)) > 0 {
                            rules_num_match += 1
                            logs.Debug("路由【%s】规则匹配成功：标签=%s, 正则值=%s, 实际值=%s", router_value.Name, rule.Name, rule.Value, label_value)
                        } else {
                            logs.Debug("路由【%s】规则匹配失败：标签=%s, 正则值=%s, 实际值=%s", router_value.Name, rule.Name, rule.Value, label_value)
                        }
                    } else {
                        // 非正则部分比对
                        if rule.Value == label_value.(string) {
                            rules_num_match += 1
                            logs.Debug("路由【%s】规则匹配成功：标签=%s, 期望值=%s, 实际值=%s", router_value.Name, rule.Name, rule.Value, label_value)
                        } else {
                            logs.Debug("路由【%s】规则匹配失败：标签=%s, 期望值=%s, 实际值=%s", router_value.Name, rule.Name, rule.Value, label_value)
                        }
                    }
                }
            }
            if !labelFound {
                logs.Debug("路由【%s】规则匹配失败：标签=%s 在告警中未找到", router_value.Name, rule.Name)
            }
        }

		//判断如果路由规则匹配，需要替换url到现有的参数中
		if rules_num == rules_num_match {
			is_matched = true
			// 检查是否发送告警，如果SendAlert为false，则抑制该告警
			if router_value.SendAlert == false {
				logs.Info("告警路由【%s】匹配成功，设置为不发送，抑制该告警。", router_value.Name)
				return []PrometheusAlertMsg{} // 返回空切片，抑制所有消息
			}

			logs.Info("告警路由【%s】匹配成功，准备发送告警", router_value.Name)
			newMsg := PMsg // 创建一个副本以防修改原始PMsg
			newMsg.Type = router_value.Tpl.Tpltype
			newMsg.Tpl = router_value.Tpl.Tpl
			atSomeOne := router_value.AtSomeOne
			if router_value.AtSomeOneRR {
				openIds := strings.Split(router_value.AtSomeOne, ",")
				if len(openIds) > 1 {
					// 用自1970年1月1日以来的天数取余计算
					duration := time.Since(time.Unix(0, 0))
					days := duration.Hours() / 24
					i := int(days) % len(openIds)
					atSomeOne = openIds[i]
				}
			}

			switch router_value.Tpl.Tpltype {
			case "wx":
				newMsg.Wxurl = router_value.UrlOrPhone
				newMsg.AtSomeOne = atSomeOne
			//钉钉渠道
			case "dd":
				newMsg.Ddurl = router_value.UrlOrPhone
				newMsg.AtSomeOne = atSomeOne
			//飞书渠道
			case "fs":
				newMsg.Fsurl = router_value.UrlOrPhone
				newMsg.AtSomeOne = atSomeOne
			//Webhook渠道
			case "webhook":
				newMsg.WebHookUrl = router_value.UrlOrPhone
			//邮件
			case "email":
				newMsg.Email = router_value.UrlOrPhone
			//百度Hi(如流)
			case "rl":
				newMsg.GroupId = router_value.UrlOrPhone
			//短信、电话
			case "txdx", "hwdx", "bddx", "alydx", "txdh", "alydh", "rlydh", "7moordx", "7moordh":
				newMsg.Phone = router_value.UrlOrPhone
			//异常参数
			default:
				logs.Info("暂未支持的路由！")
			}
			//匹配路由完成加入返回列表
			return_Msgs = append(return_Msgs, newMsg)

		} else {
            logs.Info("告警路由【%s】匹配失败，规则总数=%d, 匹配成功数=%d", router_value.Name, rules_num, rules_num_match)
        }

    }
	// 如果没有任何路由匹配，则使用默认模板发送
	if !is_matched {
		PMsg.Tpl = Tpl
		return_Msgs = append(return_Msgs, PMsg)
	}
	return return_Msgs
}

// 处理告警记录
func SetRecord(AlertValue interface{}) {
	var Alertname, Status, Level, Labels, Instance, Summary, Description, StartAt, EndAt string
	xalert := AlertValue.(map[string]interface{})
	PCstTime, _ := beego.AppConfig.Int("prometheus_cst_time")
	StartAt = xalert["startsAt"].(string)
	EndAt = xalert["endsAt"].(string)
	if PCstTime == 1 {
		StartAt = GetCSTtime(xalert["startsAt"].(string))
		EndAt = GetCSTtime(xalert["endsAt"].(string))
	}

	Status = xalert["status"].(string)
	//get labels

	//get alertname
	if xalert["labels"].(map[string]interface{})["alertname"] != nil {
		Alertname = xalert["labels"].(map[string]interface{})["alertname"].(string)
	}
	if xalert["labels"].(map[string]interface{})["level"] != nil {
		Level = xalert["labels"].(map[string]interface{})["level"].(string)
	}
	if xalert["labels"].(map[string]interface{})["instance"] != nil {
		Instance = xalert["labels"].(map[string]interface{})["instance"].(string)
	}
	labelsJsonStr, err := json.Marshal(xalert["labels"].(map[string]interface{}))
	if err != nil {
		logs.Error("转换lables失败：", err)
	} else {
		Labels = string(labelsJsonStr)
	}

	//get description
	if xalert["annotations"].(map[string]interface{})["description"] != nil {
		Description = xalert["annotations"].(map[string]interface{})["description"].(string)
	}
	//get summary
	if xalert["annotations"].(map[string]interface{})["summary"] != nil {
		Summary = xalert["annotations"].(map[string]interface{})["summary"].(string)
	}

	if beego.AppConfig.String("AlertRecord") == "1" && !models.GetRecordExist(Alertname, Level, Labels, Instance, StartAt, EndAt, Summary, Description, Status) {
		models.AddAlertRecord(Alertname,
			Level,
			Labels,
			Instance,
			StartAt,
			EndAt,
			Summary,
			Description,
			Status)
	}

	// 告警写入ES
	if beego.AppConfig.DefaultString("alert_to_es", "0") == "1" {
		dt := time.Now()
		dty, dtm := dt.Year(), int(dt.Month())
		esIndex := fmt.Sprintf("prometheusalert-%d%02d", dty, dtm)
		alert := &elastic.AlertES{
			Alertname:   Alertname,
			Status:      Status,
			Instance:    Instance,
			Level:       Level,
			Labels:      Labels,
			Summary:     Summary,
			Description: Description,
			StartsAt:    StartAt,
			EndsAt:      EndAt,
			Created:     dt,
		}
		go elastic.Insert(esIndex, *alert)
	}
}

// 消息模版化
func TransformAlertMessage(p_json interface{}, tpltext string) (error error, msg string) {
	funcMap := template.FuncMap{
		"GetTimeDuration": GetTimeDuration,
		"GetCSTtime":      GetCSTtime,
		"TimeFormat":      TimeFormat,
		"GetTime":         GetTime,
		"toUpper":         strings.ToUpper,
		"toLower":         strings.ToLower,
		"title":           strings.Title,
		// join is equal to strings.Join but inverts the argument order
		// for easier pipelining in templates.
		"join": func(sep string, s []string) string {
			return strings.Join(s, sep)
		},
		"match": regexp.MatchString,
		"safeHtml": func(text string) tmplhtml.HTML {
			return tmplhtml.HTML(text)
		},
		"reReplaceAll": func(pattern, repl, text string) string {
			re := regexp.MustCompile(pattern)
			return re.ReplaceAllString(text, repl)
		},
		"stringSlice": func(s ...string) []string {
			return s
		},
		"SplitString": func(pstring string, start int, stop int) string {
			beego.Debug("SplitString", pstring)
			if stop < 0 {
				return pstring[start : len(pstring)+stop]
			}
			return pstring[start:stop]
		},
	}

	buf := new(bytes.Buffer)
	tpl, err := template.New("").Funcs(funcMap).Parse(tpltext)
	if err != nil {
		return err, ""
	}

	err = tpl.Execute(buf, p_json)
	if err != nil {
		return err, ""
	}

	return nil, buf.String()
}

// 发送消息
func SendMessagePrometheusAlert(message string, pmsg *PrometheusAlertMsg, logsign string) string {
	Title := beego.AppConfig.String("title")
	var ReturnMsg string
	models.AlertsFromCounter.WithLabelValues("/prometheusalert").Add(1)
	ChartsJson.Prometheusalert += 1
	switch pmsg.Type {
	//微信渠道
	case "wx":
		Wxurl := strings.Split(pmsg.Wxurl, ",")
		if pmsg.RoundRobin == "true" {
			ReturnMsg += PostToWeiXin(message, DoBalance(Wxurl), pmsg.AtSomeOne, logsign)
		} else {
			for _, url := range Wxurl {
				ReturnMsg += PostToWeiXin(message, url, pmsg.AtSomeOne, logsign)
			}
		}

	//钉钉渠道
	case "dd":
		Ddurl := strings.Split(pmsg.Ddurl, ",")
		if pmsg.RoundRobin == "true" {
			ReturnMsg += PostToDingDing(Title, message, DoBalance(Ddurl), pmsg.AtSomeOne, logsign)
		} else {
			for _, url := range Ddurl {
				ReturnMsg += PostToDingDing(Title, message, url, pmsg.AtSomeOne, logsign)
			}
		}

	//飞书渠道
	case "fs":
		Fsurl := strings.Split(pmsg.Fsurl, ",")
		if pmsg.RoundRobin == "true" {
			ReturnMsg += PostToFS(Title, message, DoBalance(Fsurl), pmsg.AtSomeOne, logsign)
		} else {
			for _, url := range Fsurl {
				ReturnMsg += PostToFS(Title, message, url, pmsg.AtSomeOne, logsign)
			}
		}

	//Webhook渠道
	case "webhook":
		Fwebhookurl := strings.Split(pmsg.WebHookUrl, ",")
		if pmsg.RoundRobin == "true" {
			ReturnMsg += PostToWebhook(message, DoBalance(Fwebhookurl), logsign, pmsg.WebhookContentType)
		} else {
			for _, url := range Fwebhookurl {
				ReturnMsg += PostToWebhook(message, url, logsign, pmsg.WebhookContentType)
			}
		}

	//腾讯云短信
	case "txdx":
		ReturnMsg += PostTXmessage(message, pmsg.Phone, logsign)
	//华为云短信
	case "hwdx":
		ReturnMsg += PostHWmessage(message, pmsg.Phone, logsign)
	//百度云短信
	case "bddx":
		ReturnMsg += PostBDYmessage(message, pmsg.Phone, logsign)
	//阿里云短信
	case "alydx":
		ReturnMsg += PostALYmessage(message, pmsg.Phone, logsign)
	//腾讯云电话
	case "txdh":
		ReturnMsg += PostTXphonecall(message, pmsg.Phone, logsign)
	//阿里云电话
	case "alydh":
		ReturnMsg += PostALYphonecall(message, pmsg.Phone, logsign)
	//容联云电话
	case "rlydh":
		ReturnMsg += PostRLYphonecall(message, pmsg.Phone, logsign)
	//七陌短信
	case "7moordx":
		ReturnMsg += Post7MOORmessage(message, pmsg.Phone, logsign)
	//七陌语音电话
	case "7moordh":
		ReturnMsg += Post7MOORphonecall(message, pmsg.Phone, logsign)
	//邮件
	case "email":
		ReturnMsg += SendEmail(message, pmsg.Email, pmsg.EmailTitle, logsign)
	// Telegram
	case "tg":
		ReturnMsg += SendTG(message, logsign)
	// Workwechat
	case "workwechat":
		ReturnMsg += SendWorkWechat(pmsg.ToUser, pmsg.ToParty, pmsg.ToTag, message, logsign)
	//百度Hi(如流)
	case "rl":
		ReturnMsg += PostToRuLiu(pmsg.GroupId, message, beego.AppConfig.String("BDRL_URL"), logsign)
	// Bark
	case "bark":
		ReturnMsg += SendBark(message, logsign)
	// Bark
	case "voice":
		ReturnMsg += SendVoice(message, logsign)
	//飞书APP渠道
	case "fsapp":
		ReturnMsg += PostToFeiShuApp(Title, message, pmsg.AtSomeOne, logsign)
	//kafka渠道
	case "kafka":
		ReturnMsg += SendKafka(message, logsign)
	//异常参数
	default:
		ReturnMsg = "参数错误"
	}
	return ReturnMsg
}
