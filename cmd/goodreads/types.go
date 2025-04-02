package goodreads

// Book struct represents a book entry in the CSV
type Book struct {
	ID                       int      `json:"Book Id"`
	Title                    string   `json:"Title"`
	Authors                  []string `json:"Authors"`
	ISBN                     string   `json:"ISBN"`
	ISBN13                   string   `json:"ISBN13"`
	MyRating                 float64  `json:"My Rating"`
	AverageRating            float64  `json:"Average Rating"`
	Publisher                string   `json:"Publisher"`
	Binding                  string   `json:"Binding"`
	NumberOfPages            int      `json:"Number of Pages"`
	YearPublished            int      `json:"Year Published"`
	OriginalPublicationYear  int      `json:"Original Publication Year"`
	DateRead                 string   `json:"Date Read"`
	DateAdded                string   `json:"Date Added"`
	Bookshelves              []string `json:"Bookshelves"`
	BookshelvesWithPositions []string `json:"Bookshelves with positions"`
	ExclusiveShelf           string   `json:"Exclusive Shelf"`
	MyReview                 string   `json:"My Review"`
	Spoiler                  string   `json:"Spoiler"`
	PrivateNotes             string   `json:"Private Notes"`
	ReadCount                int      `json:"Read Count"`
	OwnedCopies              int      `json:"Owned Copies"`
	Description              string   `json:"Description"`
	Subjects                 []string `json:"Subjects"`
	CoverID                  int      `json:"Cover ID"`
	CoverURL                 string   `json:"Cover URL"`
	SubjectPeople            []string `json:"Subject People"`
	Subtitle                 string   `json:"Subtitle"`
}

// OpenLibraryBook struct represents the response from the OpenLibrary API
type OpenLibraryBook struct {
	Description   any    `json:"description"`
	Covers        []int  `json:"covers"`
	Subjects      []any  `json:"subjects"`
	SubjectPeople []any  `json:"subject_people"`
	Title         string `json:"title"`
	Subtitle      string `json:"subtitle"`
	Publishers    []struct {
		Name string `json:"name"`
	} `json:"publishers"`
	Cover struct {
		Small  string `json:"small"`
		Medium string `json:"medium"`
		Large  string `json:"large"`
	} `json:"cover"`
	Authors []struct {
		URL      string `json:"url"`
		Name     string `json:"name"`
		Key      string `json:"key"`
		Personal string `json:"personal_name,omitempty"`
	} `json:"authors"`
	PublishDate string `json:"publish_date"`
	Identifiers struct {
		ISBN         []string `json:"isbn_10,omitempty"`
		ISBN13       []string `json:"isbn_13,omitempty"`
		OCLC         []string `json:"oclc,omitempty"`
		Goodreads    []string `json:"goodreads,omitempty"`
		LibraryThing []string `json:"librarything,omitempty"`
	} `json:"identifiers"`
	NumberOfPages int    `json:"number_of_pages"`
	Weight        string `json:"weight,omitempty"`
	Links         []struct {
		URL   string `json:"url"`
		Title string `json:"title"`
	} `json:"links,omitempty"`
	Notes         string `json:"notes,omitempty"`
	PublishPlaces []struct {
		Name string `json:"name"`
	} `json:"publish_places,omitempty"`
	Excerpts []struct {
		Text    string `json:"text"`
		Comment string `json:"comment,omitempty"`
	} `json:"excerpts,omitempty"`
}

// OpenLibraryEdition struct represents the response from the OpenLibrary API
type OpenLibraryEdition struct {
	Publishers      []string       `json:"publishers"`
	Number_of_pages int            `json:"number_of_pages"`
	Cover           map[string]any `json:"cover"`
	Publish_date    string         `json:"publish_date,omitempty"`
	Title           string         `json:"title,omitempty"`
	Identifiers     struct {
		ISBN_10      []string `json:"isbn_10,omitempty"`
		ISBN_13      []string `json:"isbn_13,omitempty"`
		OCLC         []string `json:"oclc,omitempty"`
		Goodreads    []string `json:"goodreads,omitempty"`
		LibraryThing []string `json:"librarything,omitempty"`
	} `json:"identifiers,omitempty"`
	Languages []struct {
		Key string `json:"key"`
	} `json:"languages,omitempty"`
	Authors []struct {
		Key string `json:"key"`
	} `json:"authors,omitempty"`
	Works []struct {
		Key string `json:"key"`
	} `json:"works,omitempty"`
	Subjects []string `json:"subjects,omitempty"`
	Weight   string   `json:"weight,omitempty"`
	Notes    string   `json:"notes,omitempty"`
}

// OpenLibraryAuthor struct represents author data from the OpenLibrary API
type OpenLibraryAuthor struct {
	Name           string   `json:"name"`
	BirthDate      string   `json:"birth_date,omitempty"`
	DeathDate      string   `json:"death_date,omitempty"`
	Bio            any      `json:"bio,omitempty"`
	Wikipedia      string   `json:"wikipedia,omitempty"`
	Photos         []int    `json:"photos,omitempty"`
	AlternateNames []string `json:"alternate_names,omitempty"`
}

// OpenLibraryWork struct represents work data from the OpenLibrary API
type OpenLibraryWork struct {
	Title            string   `json:"title"`
	Description      any      `json:"description,omitempty"`
	Covers           []int    `json:"covers,omitempty"`
	Subjects         []string `json:"subjects,omitempty"`
	SubjectPlaces    []string `json:"subject_places,omitempty"`
	SubjectTimes     []string `json:"subject_times,omitempty"`
	SubjectPeople    []string `json:"subject_people,omitempty"`
	FirstPublishDate string   `json:"first_publish_date,omitempty"`
	Authors          []struct {
		Author struct {
			Key string `json:"key"`
		} `json:"author"`
		Type struct {
			Key string `json:"key"`
		} `json:"type,omitempty"`
	} `json:"authors,omitempty"`
}
