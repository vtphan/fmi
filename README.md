#### func  Load

```go
func Load(dir string) *Index
```
-----------------------------------------------------------------------------
Load FM index. Usage: idx := Load(index_file)

#### func  New

```go
func New(file string) *Index
```
-----------------------------------------------------------------------------
Build FM index given the file storing the text.

#### func (*Index) Repeat

```go
func (I *Index) Repeat(j, read_len int) []int
```
Search for all repeats of SEQ[j:j+read_len] in SEQ

#### func (*Index) Save

```go
func (I *Index) Save(dirname string)
```
-----------------------------------------------------------------------------
Save the index to directory.

#### func (*Index) Search

```go
func (I *Index) Search(pattern []byte) []int
```
Search for all occurences of pattern in SEQ

#### func (*Index) SearchFrom

```go
func (I *Index) SearchFrom(pattern []byte, start_pos int) (int, int, int)
```
Returns starting, ending positions (sp, ep) and last-matched position (i)

#### func (*Index) Show

```go
func (I *Index) Show()
```
-----------------------------------------------------------------------------
