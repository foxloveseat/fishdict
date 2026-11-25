package deftypes

type Trip struct {
	Id      int
	Date    string
	Place   string
	User    User
	Catches []Catch
}

type Catch struct {
	Id     int
	TripId int
	Fish   Fish
	Count  int
	Lure   Lure
}

type Fish struct {
	Id   int
	Name string
}

type User struct {
	Id   int
	Name string
}

type Lure struct {
	Id   int
	Name string
}

func CreateUser(id int, name string) User {
	return User{
		Id:   id,
		Name: name,
	}
}

func CreateLure(id int, devel string, name string) Lure {
	return Lure{
		Id:   id,
		Name: name,
	}
}

func CreateFish(id int, name string) Fish {
	return Fish{
		Id:   id,
		Name: name,
	}
}
