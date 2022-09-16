package list

type foo struct {
	value      int
	next, prev *foo
}

type bar struct {
	value int
	ptr   list
}

type list struct {
	next, prev *list
}

func addToTail(node, head *list) {
	listAdd(node, head.prev, head)
}

func listAdd(node, prev, next *list) {
	if !listAddValid(node, prev, next) {
		return
	}

	next.prev = node
	node.next = next
	node.prev = prev
	writeOnce(prev.next, node)
}

func listAddValid(node, prev, next *list) bool { return true }

func writeOnce(x, val any) {
	x = val
}

func listForEach(pos, head *list, f func()) {
	for pos = head.next; !listIsHead(pos, head); pos = pos.next {
		f()
	}

	// for (pos = (head)->next; !list_is_head(pos, (head)); pos = pos->next)
}

func listIsHead(list, head *list) bool { return list == head }

/*
static inline void list_add_tail(struct list_head *new, struct list_head *head)
{
__list_add(new, head->prev, head);
}

static inline void __list_add(struct list_head *new,
struct list_head *prev,
struct list_head *next)
{
if (!__list_add_valid(new, prev, next))
return;
next->prev = new;
new->next = next;
new->prev = prev;
WRITE_ONCE(prev->next, new);
}

extern bool __list_add_valid(struct list_head *new,struct list_head *prev,	struct list_head *next);
extern bool __list_del_entry_valid(struct list_head *entry);
#else
static inline bool __list_add_valid(struct list_head *new,
struct list_head *prev,
struct list_head *next)
{
return true;
}

static inline int list_is_head(const struct list_head *list, const struct list_head *head)
{
return list == head;
}

#define list_for_each_entry(pos, head, member)				\
	for (pos = list_first_entry(head, typeof(*pos), member);	\
	     !list_entry_is_head(pos, head, member);			\
	     pos = list_next_entry(pos, member))

#define list_entry_is_head(pos, head, member)				\
	(&pos->member == (head))

	#define list_next_entry(pos, member) \
	list_entry((pos)->member.next, typeof(*(pos)), member)

	#define list_entry(ptr, type, member) \
	container_of(ptr, type, member)
*/
