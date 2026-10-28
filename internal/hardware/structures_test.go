package hardware

import "testing"

func TestWordReadsALittleEndianWord(t *testing.T) {
	data := []byte{0xA0, 0x0F, 0x40, 0x00}

	if got := Word(data, 0); got != 0x0FA0 {
		t.Errorf("第一个数是 %d，不是 %d", got, 0x0FA0)
	}
	if got := Word(data, 2); got != 0x0040 {
		t.Errorf("第二个数是 %d，不是 %d", got, 0x0040)
	}
}

func TestWordOfAByteThatIsNotThere(t *testing.T) {
	if got := Word([]byte{0x01}, 0); got != -1 {
		t.Errorf("读不到的时候该报 -1，是 %d", got)
	}
	if got := Word(nil, 0); got != -1 {
		t.Errorf("读不到的时候该报 -1，是 %d", got)
	}
}

func TestStructuresWalksTheTable(t *testing.T) {
	first := make([]byte, 6)
	first[0] = 4
	first[1] = 6

	second := make([]byte, 8)
	second[0] = 17
	second[1] = 8

	table := []byte{}
	table = append(table, first...)
	table = append(table, 0x41, 0x42, 0x00, 0x00)
	table = append(table, second...)
	table = append(table, 0x00, 0x00)

	structures := Structures(table)
	if len(structures) != 2 {
		t.Fatalf("表里有 %d 段，不是两段", len(structures))
	}
	if structures[0].Type != 4 || len(structures[0].Data) != 6 {
		t.Errorf("第一段不对: %#v", structures[0])
	}
	if structures[1].Type != 17 || len(structures[1].Data) != 8 {
		t.Errorf("第二段不对: %#v", structures[1])
	}
	if len(structures[0].Strings) == 0 {
		t.Errorf("第一段后面没有字符串: %#v", structures[0].Strings)
	}
}

func TestStructuresOfATableThatEndsTooSoon(t *testing.T) {
	table := []byte{4, 40, 0, 0}
	if structures := Structures(table); len(structures) != 0 {
		t.Errorf("不该读出结构: %#v", structures)
	}

	if structures := Structures([]byte{4, 2}); len(structures) != 0 {
		t.Errorf("长度不对的时候不该读出结构: %#v", structures)
	}
}
