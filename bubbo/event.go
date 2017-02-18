package bubbo

type Detail map[string]interface{}

type Event struct{
	Group string
	Type string
	Detail Detail
}

func errorEvent(err error)*Event{
	return &Event{
		Group:"*",
		Type:"err",
		Detail:Detail{"msg":err.Error()},
	}
}

type EventHandler func(*Event)(*Event,error)
