package bubbo

import "github.com/asdine/storm"

type Media struct {
	Name string `storm:"id"`
	Data string
}

func media(name string) (m *Media, err error) {
	m = new(Media)
	err = db.One("Name", name, m)
	if err == storm.ErrNotFound {
		err = ErrorMediaNotFound
	}
	return
}

func (u *User)saveMedia()(err error){
	return nil
}