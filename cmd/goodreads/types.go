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
}

// OpenLibraryEdition struct represents the response from the OpenLibrary API
type OpenLibraryEdition struct {
	Publishers      []string       `json:"publishers"`
	Number_of_pages int            `json:"number_of_pages"`
	Cover           map[string]any `json:"cover"`
}
