const utils = {
    isElementCompletelyInViewport (el) {
        const rect = el.getBoundingClientRect()
        const windowHeight = window.innerHeight || document.documentElement.clientHeight
        const windowWidth = window.innerWidth || document.documentElement.clientWidth
        return (
            rect.top >= 0 &&
            rect.left >= 0 &&
            rect.bottom <= (window.innerHeight || document.documentElement.clientHeight) &&
            rect.right <= (window.innerWidth || document.documentElement.clientWidth)
        )
    },

    isElementPartiallyInViewport (el) {
        const rect = el.getBoundingClientRect()
        const windowHeight = window.innerHeight || document.documentElement.clientHeight
        const windowWidth = window.innerWidth || document.documentElement.clientWidth
        return (
            rect.bottom >= 0 &&
            rect.right >= 0 &&
            rect.top <= windowHeight &&
            rect.left <= windowWidth
        )
    },

    // https://github.com/component/debounce
    debounce(func, wait, immediate) {
        var timeout, args, context, timestamp, result;
        if (null == wait) wait = 100;
        
        function later() {
            var last = Date.now() - timestamp;
        
            if (last < wait && last >= 0) {
            timeout = setTimeout(later, wait - last);
            } else {
            timeout = null;
            if (!immediate) {
                result = func.apply(context, args);
                context = args = null;
            }
            }
        };
        
        var debounced = function(){
            context = this;
            args = arguments;
            timestamp = Date.now();
            var callNow = immediate && !timeout;
            if (!timeout) timeout = setTimeout(later, wait);
            if (callNow) {
            result = func.apply(context, args);
            context = args = null;
            }
        
            return result;
        };
        
        debounced.clear = function() {
            if (timeout) {
            clearTimeout(timeout);
            timeout = null;
            }
        };
        
        debounced.flush = function() {
            if (timeout) {
            result = func.apply(context, args);
            context = args = null;
            
            clearTimeout(timeout);
            timeout = null;
            }
        };
        
        return debounced;
        }

}

const api = {
    search(queryString) {
        const query = queryString ? { query: queryString } : { match_all: {} }
        return fetch('/api/search', {
            method: 'POST',
            body: JSON.stringify({
                size: 2000,
                /*
                facets: {
                    tags: {
                        size: 10,
                        field: 'XMP:Subject'
                    },
                    rating: {
                        size: 10,
                        field: 'XMP:Rating',
                        numeric_ranges: [
                            {
                                name: 'everything',
                                min: 0
                            },
                            {
                                name: '1+',
                                min: 1,
                            },
                            {
                                name: '2+',
                                min: 2
                            },
                            {
                                name: '3+',
                                min: 3
                            }
                        ]
                    }
                },
                */
                /*
                highlight: {
                    style: 'html',
                    fields: [
                        'XMP:Subject',
                        'XMP:Rating' 
                    ]
                },
                */
                query: query,
                fields: ['XMP:Subject', 'XMP:Rating'],
                sort: ['_id']
            })
        }).then(response => response.json())
    }
}

new Vue({
    el: '#app',
    data() {
        return {
            query: '',
            hits: [],
            hitIdxById: {},
            selectedId: null,
            atLeastOnceVisibleIdxs: [],
            currentlyVisibleIdxs: [],
            lastMinVisibleIdx: -1,
            lastMaxVisibleIdx: -1
        }
    },
    watch: {
        query() {
            this.search()
        }
    },
    mounted() {
        this.search()

        window.addEventListener('keydown', e => {
            const keyMap = {
                37: 'left',
                39: 'right',
                40: 'down',
                38: 'up'
            }
            if (e.keyCode in keyMap) {
                e.preventDefault()
                e.stopPropagation()
                this.$emit('cursor', keyMap[e.keyCode])
            }
        }, true)

        this.$on('cursor', key => {
            let idx = -1
            if (this.selectedId === null) {
                if (this.hits.length > 0) {
                    idx = 0
                }
            } else {
                idx = this.hitIdxById[this.selectedId]
                if (key === 'right') {
                    idx++
                } else if (key === 'left') {
                    idx--
                } else if (key === 'down') {
                    idx += this.getColumnCount()
                } else if (key === 'up') {
                    idx -= this.getColumnCount()
                }
            }

            if (idx >= this.hits.length || idx < 0) return

            if (idx !== -1 && idx < this.hits.length) {
                this.selectedId = this.hits[idx].id
            } else {
                this.selectedId = null
            }
            Vue.nextTick(() => {
                if (this.$refs.hits.length > 0) {
                    const el = this.$refs.hits[idx]
                    if (!utils.isElementCompletelyInViewport(el)) {
                        el.scrollIntoView(key === 'up' || key === 'left')
                    }
                }
            })
        })

        const getScrollTop = () => window.pageYOffset || document.documentElement.scrollTop
        

        let previousScrollTop = getScrollTop()
        window.addEventListener('scroll', utils.debounce(() => {
            let scrollTop = getScrollTop()
            this.checkVisible(scrollTop > previousScrollTop)
            previousScrollTop = scrollTop
        }))
    },
    computed: {
        selected () {
            if (!this.selectedId) return
            return this.hits[this.hitIdxById[this.selectedId]]
        }
    },
    methods: {
        getColumnCount () {
            const grid = document.querySelector('.grid')
            const gridItem = document.querySelector('.grid-item')
            return Math.round(grid.offsetWidth / gridItem.offsetWidth)
        },
        checkVisible (isDown = true) {
            const hitEls = this.$refs.hits
            let checked = 0
            if (hitEls && hitEls.length > 0) {
                const columnCount = this.getColumnCount()

                let minVisibleIdx = -1
                let maxVisibleIdx = -1
                let foundVisible = false

                if (isDown) {

                    // downward scroll

                    const startIdx = this.lastMaxVisibleIdx !== -1 ? this.lastMaxVisibleIdx : 0
                    //const startIdx = this.lastMinVisibleIdx !== -1 ? this.lastMinVisibleIdx : 0

                    for (let idx = startIdx; idx < hitEls.length; idx += columnCount) {
                        let el = hitEls[idx]
                        checked++
                        if (utils.isElementPartiallyInViewport(el)) {
                            if (!foundVisible) {
                                minVisibleIdx = idx
                            }
                            foundVisible = true
                        } else {
                            if (foundVisible) {
                                maxVisibleIdx = idx - 1
                                break
                            }
                        }
                    }
                    if (minVisibleIdx !== -1 && maxVisibleIdx === -1) maxVisibleIdx = hitEls.length - 1
                } else {
                    // upward scroll!

                    const startIdx = this.lastMinVisibleIdx !== -1 ? this.lastMinVisibleIdx : hitEls.length - 1
                    //const startIdx = this.lastMaxVisibleIdx !== -1 ? this.lastMaxVisibleIdx : hitEls.length - 1

                    for (let idx = startIdx; idx > 0; idx -= columnCount) {
                        let el = hitEls[idx]
                        checked++
                        if (utils.isElementPartiallyInViewport(el)) {
                            if (!foundVisible) {
                                maxVisibleIdx = idx
                            }
                            foundVisible = true
                        } else {
                            if (foundVisible) {
                                minVisibleIdx = idx - 1
                                break
                            }
                        }
                    }
                    if (maxVisibleIdx !== -1 && minVisibleIdx === -1) minVisibleIdx = 0
                }

                if (minVisibleIdx !== -1 && maxVisibleIdx !== -1) {
                    const currentlyVisibleIdxs = []
                    for (let idx = minVisibleIdx; idx <= maxVisibleIdx; idx++) {
                        Vue.set(this.atLeastOnceVisibleIdxs, idx, true)
                        currentlyVisibleIdxs[idx] = true
                    }
                    this.currentlyVisibleIdxs = currentlyVisibleIdxs
                }

                this.lastMinVisibleIdx = minVisibleIdx
                this.lastMaxVisibleIdx = maxVisibleIdx
            }
        },
        search() {
            api.search(this.query).then(result => {
                this.hits = result.hits
                this.hitIdxById = {}
                for (let i = 0; i < this.hits.length; i++) {
                    this.hitIdxById[this.hits[i].id] = i
                }
                Vue.nextTick(() => {
                    this.checkVisible()
                })
            })
        },
        thumbnailSrc (id) {
            return `/images/320x240/${id}`
        },
        select (id) {
            this.selectedId = id
        },
        tagsFor (hit) {
            const val = hit.fields['XMP:Subject']
            if (!val) return []
            return val.split(',')
        },
        isVisible (idx) {
            //return this.currentlyVisibleIdxs[idx]
            return this.atLeastOnceVisibleIdxs[idx]
        }
    },
    template: `
        <div>
            <input type="text" v-model="query">
            <foo></foo>
            <div class="grid">
                <div
                    class="grid-item"
                    v-for="(hit, idx) in hits"
                    :key="hit.id"
                    :class="{ 'selected': hit.id === selectedId }"
                    ref="hits">
                    <div
                        @click="select(hit.id)"
                        class="hit">
                        <img :src="isVisible(idx) ? thumbnailSrc(hit.id) : ''">
                    </div>
                    <div class="meta">
                        <span class="id">{{ hit.id }}</span>
                        <span v-for="tag in tagsFor(hit)">
                            <span class="tag">{{ tag }}</span>
                        </span>
                    </div>
                </div>
            </div>
        </div>
    `
})