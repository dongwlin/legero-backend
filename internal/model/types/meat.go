package types

type Meat string

const (
	MeatNone            Meat = "none"            // 无
	MeatPorkLean        Meat = "PorkLean"        // 瘦肉
	MeatPorkLiver       Meat = "PorkLiver"       // 猪肝
	MeatPorkBlood       Meat = "PorkBlood"       // 猪血
	MeatPorkIntestineLg Meat = "PorkIntestineLg" // 大肠
	MeatPorkIntestineSm Meat = "PorkIntestineSm" // 小肠
	MeatPorkKidney      Meat = "PorkKidney"      // 猪腰
)

func (m Meat) String() string {
	return string(m)
}

func MeatsToStrings(ms []Meat) []string {
	strs := make([]string, 0, len(ms))
	for _, m := range ms {
		strs = append(strs, m.String())
	}
	return strs
}

func StringsToMeats(strs []string) []Meat {
	ms := make([]Meat, 0, len(strs))
	for _, str := range strs {
		ms = append(ms, Meat(str))
	}
	return ms
}
