package bubbo

type Detail map[string]interface{}

type Event struct{
	Group string
	Type string
	Detail Detail
}

func errorEvent(group,_type string,err error)*Event{
	return &Event{
		Group:group,
		Type:_type,
		Detail:Detail{"error":true,"msg":err.Error()},
	}
}

type EventHandler func(*Event)(*Event,error)
