package goodreads

// Book struct represents a book entry in the CSV
type Book struct {
	ID                       int      `json:"bookId"`
	Title                    string   `json:"title"`
	Authors                  []string `json:"authors"`
	ISBN                     string   `json:"isbn"`
	ISBN13                   string   `json:"isbn13"`
	MyRating                 float64  `json:"myRating"`
	AverageRating            float64  `json:"averageRating"`
	Publisher                string   `json:"publisher"`
	Binding                  string   `json:"binding"`
	NumberOfPages            int      `json:"numberOfPages"`
	YearPublished            int      `json:"yearPublished"`
	OriginalPublicationYear  int      `json:"originalPublicationYear"`
	DateRead                 string   `json:"dateRead"`
	DateAdded                string   `json:"dateAdded"`
	Bookshelves              []string `json:"bookshelves"`
	BookshelvesWithPositions []string `json:"bookshelvesWithPositions"`
	ExclusiveShelf           string   `json:"exclusiveShelf"`
	MyReview                 string   `json:"myReview"`
	Spoiler                  string   `json:"spoiler"`
	PrivateNotes             string   `json:"privateNotes"`
	ReadCount                int      `json:"readCount"`
	OwnedCopies              int      `json:"ownedCopies"`
	Description              string   `json:"description"`
	Subjects                 []string `json:"subjects"`
	CoverID                  int      `json:"coverId"`
	CoverURL                 string   `json:"coverUrl"`
	SubjectPeople            []string `json:"subjectPeople"`
	Subtitle                 string   `json:"subtitle"`
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

// GoogleBooksBook represents a single item from the Google Books API response
type GoogleBooksBook struct {
	VolumeInfo struct {
		Title               string   `json:"title"`
		Subtitle            string   `json:"subtitle"`
		Authors             []string `json:"authors"`
		Publisher           string   `json:"publisher"`
		PublishedDate       string   `json:"publishedDate"`
		Description         string   `json:"description"`
		IndustryIdentifiers []struct {
			Type       string `json:"type"`
			Identifier string `json:"identifier"`
		} `json:"industryIdentifiers"`
		PageCount     int      `json:"pageCount"`
		Categories    []string `json:"categories"`
		AverageRating float64  `json:"averageRating"`
		RatingsCount  int      `json:"ratingsCount"`
		ImageLinks    struct {
			Thumbnail      string `json:"thumbnail"`
			SmallThumbnail string `json:"smallThumbnail"`
		} `json:"imageLinks"`
		Language string `json:"language"`
		InfoLink string `json:"infoLink"`
	} `json:"volumeInfo"`
}

// GoogleBooksResponse represents the API response wrapper from Google Books
type GoogleBooksResponse struct {
	TotalItems int               `json:"totalItems"`
	Items      []GoogleBooksBook `json:"items"`
}

// CachedOpenLibraryBook wraps OpenLibraryBook with metadata for negative caching
type CachedOpenLibraryBook struct {
	Book     *OpenLibraryBook `json:"book"`
	NotFound bool             `json:"not_found"`
}

// CachedGoogleBooksBook wraps GoogleBooksBook with metadata for negative caching
type CachedGoogleBooksBook struct {
	Book     *GoogleBooksBook `json:"book"`
	NotFound bool             `json:"not_found"`
}
