package main

//SQLTableCreation stores the SQL table creation statements
var SQLTableCreation = []string{
	`CREATE TABLE cards (
			"cardCode" TEXT NOT NULL PRIMARY KEY,
			"set" TEXT,		
			"regionRef" TEXT,
			"attack" INTEGER,
			"health" INTEGER,
			"cost" INTEGER,
			"artistname" TEXT,
			"spellSpeedRef" TEXT,
			"rarityRef" TEXT,
			"subtype" TEXT,
			"subtype2" TEXT,
			"supertype" TEXT,
			"type" TEXT,
			"collectible" INTEGER		
		);`,
	`CREATE TABLE keywords (
			"cardCode" TEXT,		
			"keyword" TEXT,
			PRIMARY KEY ("cardCode", "keyword")		
		);`,
	`CREATE TABLE associations (
			"cardCode" TEXT,		
			"associated" TEXT,
			PRIMARY KEY ("cardCode", "associated")		
		);`,
	`CREATE TABLE localeCards (
			"cardCode" TEXT,		
			"locale" TEXT,
			"name" TEXT,
			"description" TEXT,
			"descriptionRaw" TEXT,
			"levelupDescription" TEXT,
			"levelupDescriptionRaw" TEXT,
			PRIMARY KEY ("cardCode", "locale")		
		);`,
	`CREATE TABLE localeGeneric (
			"type" TEXT,
			"locale" TEXT,	
			"nameRef" TEXT,
			"name" TEXT,
			"description" TEXT,
			PRIMARY KEY ("type", "locale", "nameRef")		
		);`}

//InsertCardSQLQ insert into cards
var InsertCardSQLQ = `INSERT INTO cards(
		cardCode,
		"set",
		regionRef,
		attack,
		health,
		cost,
		artistname,
		spellSpeedRef,
		rarityRef,
		subtype,
		subtype2,
		supertype,
		type,
		collectible 
	) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

//InsertKeywordsSQLQ insert into keywords
var InsertKeywordsSQLQ = `INSERT INTO keywords(
		cardCode,
		keyword
	) VALUES ( ?, ?)`

//InsertAssociationsSQLQ insert into keywords
var InsertAssociationsSQLQ = `INSERT INTO associations(
	cardCode,
	associated
) VALUES ( ?, ?)`

//InsertLocaleCardsSQLQ insert into localeCards
var InsertLocaleCardsSQLQ = `INSERT INTO localeCards(
	cardCode,
	locale,
	name,
	description,
	descriptionRaw,
	levelupDescription,
	levelupDescriptionRaw
) VALUES ( ?, ?, ?, ?, ?, ?, ?)`

//InsertLocaleGenericSQLQ insert into LocaleGeneric
var InsertLocaleGenericSQLQ = `INSERT INTO localeGeneric(
	type,
	locale,
	nameRef,
	name,
	description
) VALUES ( ?, ?, ?, ?, ?)`

//DDUrl url for the data dragon
var DDUrl = "https://dd.b.pvp.net/"

//Locales list of all the locales
var Locales = []string{
	"en_us",
	"de_de",
	"es_es",
	"es_mx",
	"fr_fr",
	"it_it",
	"ja_jp",
	"ko_kr",
	"pl_pl",
	"pt_br",
	"tr_tr",
	"ru_ru",
	"zh_tw"}

//SAssets Json struct for the Assets path (included in the Card struct)
type SAssets struct {
	GameAbsolutePath, FullAbsolutePath string
}

//Card Json struct of a card
type Card struct {
	Assets []SAssets
	AssociatedCards, AssociatedCardRefs,
	Keywords, KeywordRefs, Subtypes []string
	Set, Region, RegionRef,
	Description, DescriptionRaw,
	LevelupDescription, LevelupDescriptionRaw,
	FlavorText,
	ArtistName,
	Name,
	CardCode,
	SpellSpeed, SpellSpeedRef,
	Rarity, RarityRef,
	Subtype, Subtype2, Type, Supertype string
	Attack, Cost, Health int
	Collectible          bool
}

//GlobalDescr3 Json of a core definition
type GlobalDescr3 struct {
	Description, Name, NameRef string
}

//GlobalDescr2 Json of a core definition
type GlobalDescr2 struct {
	Name, NameRef string
}

//GlobalRegion Json of a core definition
type GlobalRegion struct {
	Abbreviation, IconAbsolutePath, Description, Name, NameRef string
}

//Global Json nested
type Global struct {
	VocabTerms, Keywords  []GlobalDescr3
	Regions               []GlobalRegion
	SpellSpeeds, Rarities []GlobalDescr2
}

//TypeLocale stores cardcode references to make possible a translated type
var TypeLocale map[string]string = make(map[string]string)

//SubtypeLocale stores cardcode references to make possible a translated type
var SubtypeLocale map[string]string = make(map[string]string)

//Subtype2Locale stores cardcode references to make possible a translated type
var Subtype2Locale map[string]string = make(map[string]string)

//SupertypeLocale stores cardcode references to make possible a translated type
var SupertypeLocale map[string]string = make(map[string]string)
