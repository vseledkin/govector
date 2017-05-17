package govector

import (
	"fmt"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	//m, e := NewManifold("/Volumes/data/fasttextmodel/RUEVENTOS/ru.47GB.skipgram.d128.w7.mc5.neg10.e10.b5e6")
	m, e := NewManifold("/Users/vseledkin/ru.47GB.skipgram.d128.w7.mc5.neg10.e10.b5e6")
	if e != nil {
		t.Fatal(fmt.Errorf("Invalid govector directory: [%s]", e))
	}
	e = m.Open()
	if e != nil {
		t.Fatal(fmt.Errorf("Invalid govector directory: [%s]", e))
	}
	defer m.Close()
	get := func(word string) {
		start := time.Now()
		v, e := m.GetVector(word)
		if e != nil {
			t.Fatal(fmt.Errorf("ERR Cannot get vector for: [%s] - %s", word, e))
		}
		t.Logf("%s %#v %f s.", word, v, time.Now().Sub(start).Seconds())
	}
	get("можно")
	get("Путdин")

}

func TestSplit(t *testing.T) {
	s := "Путин"
	ngrams := сomputeNGrams(s, 3, 6)
	t.Logf("%#v", ngrams)
	//if split != want {
	//t.Fatalf("Split failed want\n%s\ngot\n%s\n", want, split)
	//}
}
