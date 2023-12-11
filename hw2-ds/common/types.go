package common

// Note represents a single note
type Note struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// NoteErrorResponse is for returning error responses
type NoteErrorResponse struct {
	Msg    string `json:"msg"`
	Method string `json:"method"`
	Uri    string `json:"uri"`
	Body   string `json:"body"`
}
