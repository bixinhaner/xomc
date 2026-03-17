/* 检测节点是否可见 */
function isHiddenDom(cls){
  cls = cls || '';
  var _div = document.createElement('div');
  _div.style.width = 0;
  _div.style.height = 0;
  _div.style.position = 'absolute';
  _div.style.overflow = 'hidden';

  document.body.appendChild(_div);

  cls.split(' ').map(function(clsStr) {
    _div.classList.add(clsStr);
  })
  // 调用权限处理
  try{
    accessControl(accessJson);
  }catch(e){}
  // check dom has '.hidden' class yet
  var bool = _div.classList.contains('hidden');

  _div.remove();

  return bool;
}
function forbiddenEvent(ev){
  var target = ev.target
      tagName = target.tagName;
  
  if(tagName == 'LABEL') {
    ev.stopPropagation();
    ev.preventDefault();
    return false;
  }
}
Vue.component('el-ctable', {
  template: [
    '<div v-loading="isLoading" ref="tbctn" :style="tbStyle" :class="elCtableClass">',
      '<div class="el-ctable-toolbar" style="padding:10px 0;"><slot name="toolbar"></slot></div>',
      '<div v-show="!isCardModel" style="flex: 1 1 100%;overflow: auto;height: 100%;">',
        '<el-table ref="ctableInner" size="mini" :row-class-name="rowClassName" :show-header="showHeader" :data="frontData" :row-key="function(row){ return row[tbRowKey];}" :id="tableId" height="100%" style="width: 100%;" border stripe highlight-current-row',
          /* 事件处理 */
          '@select="select" @select-all="selectAll" @selection-change="selectionChange" @cell-mouse-enter="cellMouseEnter" @cell-mouse-leave="cellMouseLeave"',
          '@cell-click="cellClick" @cell-dblclick="cellDblclick" @row-click="rowClick" @row-contextmenu="rowContextmenu" @row-dblclick="rowDblclick" @header-click="headerClick"',
          '@sort-change="sortChange" @header-contextmenu="headerContextmenu" @filter-change="filterChange" @current-change="currentChange" @header-dragend="headerDragend" @expand-change="expandChange"',
          '>',
          '<el-table-column fixed v-if="indexable" type="index" label=" " :index="indexMethod" align="center"></el-table-column>',
          '<slot></slot>',
        '</el-table>',
      '</div>',

      /* Card list */
      '<div v-show="isCardModel" style="flex: 1 1 100%;overflow: auto;height: 100%;">',
        '<el-row class="tb-model-card">',
          '<el-col :span="5" v-for="item of frontData" style="margin: 10px; min-width: 160px;">',
            '<el-card :body-style="cardItemStyle" style="border: 1px solid #E0E4ED;border-radius: 6px;" shadow="hover">',
              /* Card title */
              '<div slot="header" class="text-ecllipsis">',
                '<el-checkbox :disabled="readonly" v-show="cardCheckable" @change="cardCheckChange" :key="item[rowKey]" :ref="item[rowKey]" :name="item[rowKey]" :value="ckList.includes(item[rowKey])"></el-checkbox> ',
                '<span v-show="!cardCheckable" :style="circleTitle"><i class="el-icon-tickets" style="color: #fff;zoom: 0.7;margin-right: 8px;vertical-align: middle;"></i></span>',
                '<el-tooltip placement="top">',
                  '<div slot="content">{{ item[cardKeys[0].label] }}</div>',
                  '<span>{{ item[cardKeys[0].field] }}</span>',
                '</el-tooltip>',
              '</div>',
              /* Card details */
              '<div class="card-details">',
                '<div v-for="(key,index) of cardKeys" v-if="index>0" class="text-ecllipsis card-item" v-html="cardFormatter(item[key.field],item,key)">',
                '</div>',
              '</div>',
              /* Operations */
              '<div style="display: flex;border-top: 1px solid #E0E4ED;">',
                '<i v-for="menu of initCardMenus(item).pre" @click="cardMenuClick(menu,item)" class="card-bt" :class="menu.cls" :style="btStyle"></i>',
                '<el-dropdown v-if="initCardMenus(item).suf.length>0" size="small">',
                  '<span class="el-dropdown-link"><i class="el-icon-more" :style="btStyle"></i></span>',
                  '<el-dropdown-menu slot="dropdown">',
                    '<el-dropdown-item v-for="sub of initCardMenus(item).suf"><i @click="cardMenuClick(sub)" :class="sub.cls" :style="btStyle"></i></el-dropdown-item>',
                  '</el-dropdown-menu>',
                '</el-dropdown>',
              '</>',
            '</el-card>',
          '</el-col>',
        '</el-row>',
      '</div>',

      /* 分页 */
      /* '<div style="padding: 3px"></div>',*/
      '<el-pagination ref="pg" @size-change="sizeChange" @current-change="currentChage" v-if="pagination" :class="{\'no-data\': total<=0,\'simple\':!showPager}"',
        ':page-sizes="pageSizes" :page-size="pageSize" :pager-count="3" :total="total" small',
        ':layout="layout">',
        '<span @click="reloadTb"><i class="el-icon el-icon-common-refresh" style="margin-left: 10px;cursor: pointer;line-height:22px;font-size:14px;"></i></span>',
      '</el-pagination>',
      '<div style="padding: 3px;" class="bottom-padding"></div>',
    '</div>'
  ].join(' '),
  props: {
    height: [Number, String],
    data: Array,
    url: String,
    queryParams: Object,
    pageSize: {
      type: Number,
      default: 50
    },
    pageList: Array,
    pagination: {
      type: Boolean,
      default: true
    },
    showPager: {
      type: Boolean,
      default: true
    },
    readonly: {
        type: Boolean,
        default: false
    },
    time: Number,
    defaultChecked: {
      type: Array,
      default: []
    },
    rowKey: String,
    id: {
      type: String,
      default: ''
    },
    rownumber: {
      type: Boolean,
      default: true
    },
    showHeader: {
      type: Boolean,
      default: true
    },
    frontPagination: {
      type: Boolean,
      default: false
    },
    filterstr: {
      type: String,
      default: ''
    },
    matchKeys: {
      type: Array,
      default: []
    },
    cardOption: {
      type: Object,
      default: function(){
        return {
          fields: [],
          menus: [],
          menuFmt: function(data,row){ return data || []; },
          checkable: false
        }
      }
    },
    type: {
      type: String,
      default: ''
    },
    limit: Number,
    composite: {
      type: Boolean,
      default: false
    },
    method: {
      type: String,
      default: 'post'
    },
    rowClassName:{
      type:Function,
      default: function(){}
    },
    dataKey:{
      type: String,
      default: ''
    }
  },
  data() {
    return {
      isLoaded: false,
      total: '',
      rows: [],
      _timer: '',
      sortName: '',
      sortOrder: '',
      sortChanges: '',
      ckList: [],
      isLoading: false,
      pageNum: this.$refs.pg?vm.$refs.pg.internalPageSize : this.pageSize*1,
      pageIndex: this.$refs.pg?vm.$refs.pg.internalCurrentPage : 1,
    }
  },
  computed: {
    tbRowKey() {
      return this.rowKey;
    },
    pagesize() {
      return this.pageSize*1;
    },
    propData() {
      return this.data;
    },
    cardCheckable(){
      return this.cardOption && this.cardOption.checkable == true;
    },
    isCardModel(){
      return this.type == 'card';
    },
    cardKeys(){// init Card key list
      var vm = this,
          list = [],
          cardFields = vm.cardOption.fields;
      
      if(cardFields) {
        cardFields.map(function(item){
          var keyObj = {};
          if(typeof item == 'string') {
            keyObj.field = item;
            keyObj.label = item;
          }else { // object params
            keyObj.field = item.field;
            keyObj.label = item.label || item.field;
            keyObj.formatter = item.formatter;
            keyObj.labelShow = item.labelShow;
          }

          list.push(keyObj);
        })
      }

      if(list.length==0) {// 
        list.push({label: '', field: ''});
      }

      return list;
    },
    cardItemStyle(){
      return {
        'min-height': '60px',
        padding: 0
      }
    },
    btStyle(){
      return {
    	height: '20px',
        padding: '6px 10px',
        flex: '1 auto',
        'text-align': 'center',
        'background-position': 'center',
        'background-repeat': 'no-repeat',
        cursor: 'pointer'
      }
    },
    circleTitle(){
      return {
        display: 'inline-block',
        width: '25px',
        height: '25px',
        overflow: 'hidden',
        'margin-right': '10px',
        'border-radius': '25px',
        'background-color': '#4D84FF',
        'vertical-align': 'middle',
        'text-align': 'center'
      }
    },
	  indexable(){
		  return this.rownumber;
	  },
    filterText(){
      return this.filterstr;
    },
    likefields(){
      return this.matchKeys;
    },
    frontData(){
      var vm = this,
          data = [];
      if(vm.tbData){
        data = vm.tbData.map(function(item){
          return item;
        });
      }
      if(vm.frontPagination == true) { // 前端分页截取数据
        var page = vm.pageIndex,
            pageSize = vm.pageNum,
            fields = vm.likefields,
            stxt = vm.filterText;
        
        if(data) {
          data = data.filter(function(row){
            // filterText
            var isMatch = false;
            if (stxt) {
              for (var key in row) {
                if (row.hasOwnProperty(key) && fields.includes(key)) {
                  if (((row[key]||'')+'').indexOf(stxt) >= 0) isMatch = true;
                }
              }
            } else isMatch = true;

            return isMatch;
          });
          if(page > Math.ceil(data.length/pageSize)) {
            page = Math.ceil(data.length/pageSize);
          }
          vm.total = data.length;
          data = data.slice((page-1)*pageSize, page*pageSize);
        }
      }
      
      return data;
    },
	  tbData() {
      // if (this.url) return this.rows;
      // else if (this.rows.length) return this.rows;
      // else return this.data;
      var vm = this,
          data = vm.data||[];

      if (vm.url && vm.rows.length) {
        if(vm.composite) {
          data = vm.unitData(data||[], vm.rows); //(data||[]).concat(vm.rows);
        }else {
          data = vm.rows;
        }
      }else if(vm.rows.length) {
    	  data = vm.rows;
      }else {
          vm.total = data.length;
      }
      
      return data;
    },
    tbHeight() {
      return this.height;
    },
    pageSizes() {
      return this.pageList ? this.pageList : [50,100,200];
    },
    params() {
      return this.queryParams ? this.queryParams : {};
    },
    tbStyle() {
      var tbH = this.height;
      if (typeof tbH == 'number') tbH += 'px';
      return {
        height: tbH? tbH : '100%',
        display: '-webkit-flex',
        display: 'flex',
        'flex-direction': 'column'
      }
    },
    elCtableClass(){
      return {
        'el-ctable': true,
        'readonly': this.readonly,
        limit: this.limit&&this.ckList.length>=this.limit
      };
    },
    layout() {
      if(this.frontPagination == true) {
        return 'total, sizes, prev, pager, next,slot';
      }else {
        if (this.showPager) {
          return 'total,sizes, prev, pager, next, jumper,slot';
        }else {
          return 'prev, jumper, next, slot';
        }
      }
      /*if (this.showPager) return 'total,sizes, prev, pager, next, jumper';
      else return 'total,sizes, prev, next, jumper';*/
    },
    pagenum() {
      var vm = this;

      return ' / '+ (Math.ceil(vm.total/vm.pageSize)||1);
    },
    needReload(){
      return this.sortChanges + this.url;
    },
    tableId(){
      return this.id;
    }
  },
  watch: {
      pagenum(val) {
        var vm = this;

        vm.$nextTick(function(){
          !vm.showPager && vm.$refs.pg.$el.querySelector('.el-pagination__jump').setAttribute('pagenum',val);
        });
      },
      propData(){
    	  var data = this.data;
          if(data && Array.isArray(data)) {
            this.rows = data;
          }else {
            this.rows = [];
          }
      },
      params:{
        handler: function(val){
          this.refresh();
        },
        deep: true
      },
      needReload() {
          this.refresh();
      },
      defaultChecked: {
        handler(newVal, oldVal) {
            var vm = this,
            	rowKey = vm.tbRowKey;
            if (rowKey) {
              vm.clearSelection();
              
              if(vm.tbData) {
                  vm.tbData.map(function(row){
                	  if(newVal.includes(row[rowKey])){
                		  vm.toggleRowSelection(row, true);
                	  }
                  });
              }
              
              newVal.map(function(item) {
                vm.ckList.push(item);
              })
            }
          },
          deep: true
      },
      filterText(){
        this.pageIndex = 1;
      },
      //监听定时刷新
      time(){
    	  var vm = this;
        if (vm.time == 0) {
          if (window['vm_tb_'+vm.id]) clearInterval(window['vm_tb_'+vm.id]);
        }else{
          window['vm_tb_'+vm.id] = setInterval(function() {
              var hasDom = false;
              
              if (vm.$refs.tbctn && vm.$refs.tbctn.parentNode) hasDom = true;
              
              if(vm.id) {
              if( document.querySelector('#'+vm.id) ) {
                
              }else {
                hasDom = false;
              }
            }

              if (vm._isDestroyed || !hasDom) clearInterval(window['vm_tb_'+vm.id]);
              else vm.refresh();
            }, vm.time * 1000);
        }
      }
  },
  methods: {
    appendCheckedRows(rows) {
        var vm = this;
    
        rows.map(function(row){
            vm.frontData.map(function(item){
                if(row[vm.tbRowKey] == item[vm.tbRowKey]) {
                    vm.$refs.ctableInner.toggleRowSelection(item, true);
                }
            });

            const curKeys = vm.frontData.map(function(item){
              return item[vm.tbRowKey];
            });
            if(!curKeys.includes(row[vm.tbRowKey])) {
              vm.$refs.ctableInner.store.states.selection.push(row);
            }
            
            if (!vm.ckList.includes(row[vm.tbRowKey])) {
                vm.ckList.push(row[vm.tbRowKey]);
            }
        });
        var selection = vm.$refs.ctableInner.store.states.selection;
        vm.$emit('selection-change', selection);
      },
      unitData(list1, list2) {
        var list = list1,
            key = this.rowKey;
        
        list2.map(function(row){
          var keys = list.map(function(m){
              return m[key];
            });
          
          if(!keys.includes(row[key])) list.push(row);
        });
        
        return list;
      },
      initCardMenus(row){
        var vm = this,
            menus = vm.cardOption.menus || [],
            obj = {
              pre: [],
              suf: []
            };
        
        // Avoid pollution source menus 
        menus = menus.map(function(item){
          return Object.assign({},item);
        })
        
        // formatter menus
        if(typeof vm.cardOption.menuFmt == 'function'){
          // set menus with the return value
          var fmenus = vm.cardOption.menuFmt(menus,row);
          if(fmenus) menus = fmenus;
        }

        // sort list
        menus = menus.sort(function(pre,suf){
          return pre.order - suf.order;
        })

        // filter unvisible
        menus = menus.filter(function(item){
          return !isHiddenDom(item.cls);
        })

        // normalize menus
        menus.map(function(item,index){
          if(index<3) {
            obj.pre.push(item);
          }else {
            obj.suf.push(item);
          }
        })

        return obj;
      },
      cardMenuClick(menuRow, row){
        var vm = this,
            click = vm.cardOption.beforeClick,
            bool = true;

        if(typeof click == 'function') {
          var hasReturn = click(menuRow, row);
          if(hasReturn === false) bool = false
        }

        bool && this.$emit('card-menu-click', menuRow, row);
      },
      cardFormatter(val,item, keyObj){
        var str = val;

        if(keyObj.formatter) {
          str = keyObj.formatter(val,item);
        }

        if(keyObj.labelShow != false) {
          str = keyObj.label + ' : ' + str;
        }
        
        return str;
      },
      cardCheckChange(bool,e){
        var vm = this,
            rowKey = vm.tbRowKey,
            Id = e.target.name,
            matchRow = [];
        
        vm.frontData.map(function(row) {
          if (Id == row[rowKey]) {
            vm.toggleRowSelection(row, bool);
            if(bool) {
              vm.ckList.push(row[rowKey]);
            }else {
              var idx = vm.ckList.indexOf(Id);
              vm.ckList.splice(idx,1);
            }
          }
        });
        vm.frontData.map(function(row) {
	        if(vm.ckList.indexOf(row[rowKey])>=0) {
	            matchRow.push(row);
	        }
        });
        vm.$emit('select',matchRow);
      },
      sleep(delay){
        var start = (new Date()).getTime();
        while((new Date()).getTime() - start < delay ) {
          continue;
        }
      },
      reset(){
    	  this.$refs.pg.internalCurrentPage = 1;
      },
      jumpPrev() {
        if(['true',true].includes(this.pagination)){
            var vm = this,
                page = vm.$refs.pg.internalCurrentPage;
            
            if(page>1) {
              vm.$refs.pg.internalCurrentPage = page - 1;
              vm.refresh();
            }
        }
      },
      refresh(ev) {
        var vm = this;
        var params = vm.params;
        if (this.pagination) {
          params = Object.assign({}, vm.params, {
            page: this.$refs.pg.internalCurrentPage,
            rows: this.$refs.pg.internalPageSize,
            sort: vm.sortName,
            order: vm.sortOrder
          });
        }
        if(vm.frontData && vm.frontData.length==0) {
          //vm.isLoading = true;
        }
        if (this.url) {
          $.ajax({
            url: this.url, //请求的url地址
            type: vm.method,
            dataType: "json", //返回格式为json
            data: params,
            success: function(req) {
              //请求成功时处理
              if (req && typeof req == 'object') {
                if(isArray(req)) {
                  	  vm.total = req.length;
                	  vm.rows = req;
                }else {
                    if(req.data){
                    	vm.total = req.data.totalRows || 0;
                      vm.rows = vm.dataKey? (req.data[vm.dataKey] || req.data.rows || []) : (req.data.rows || []);
                    }else{
		    	            vm.total = req.total || 0;
                    	vm.rows = vm.dataKey? (req[vm.dataKey] || req.rows || []) : (req.rows || []);
		                }
                }
                
                if (vm.tbRowKey) {
                  setTimeout(function() {
                    vm.rows.map(function(row) {
                      if (vm.ckList.includes(row[vm.tbRowKey])) {
                        vm.toggleRowSelection(row, true);
                      }
                    });
                  }, 50);
                }
              }else {
            	  vm.total = 0;
            	  vm.rows = [];
              }
              vm.$emit('load-success', req, ev, vm);
              setTimeout(function(){
            	  vm.isLoading = false;
                vm.isLoaded = true;
              },100);

              if(vm.rows.length == 0) {
                vm.jumpPrev();
              }
              
              vm.timerRefresh();
            },
            error: function(){
            	vm.total = 0;
            	vm.rows = [];
                setTimeout(function(){
              	  vm.isLoading = false;
                },100);
                
                vm.timerRefresh();
            }
          });
        }else {
        	setTimeout(function(){
          	  vm.isLoading = false;
            },100);
        }
      },
      timerRefresh() {
    	  var vm = this;
    	  
    	  if (vm.time) {
    	      // timer存入到window对象上以 vm_tb开头，确保定时器唯一
    	      var timerId = 'vm_tb_'+vm.id;
    	      if (window[timerId]) clearTimeout(window[timerId]);
    	      
    	      window[timerId] = setTimeout(function() {
    	        var hasDom = false;
    	        
    	        if (vm.$refs.tbctn && vm.$refs.tbctn.parentNode) hasDom = true;
    	        
    	        if(vm.id) {
    	    		if( document.querySelector('#'+vm.id) ) {
    	    			
    	    		}else {
    	    			hasDom = false;
    	    		}
    	    	}
    	        
    	        if (vm._isDestroyed || !hasDom) {
    	        	clearTimeout(window[timerId]);
    	        }else {
    	        	var visible = isVisible(document.querySelector('#'+vm.id)),
    	        		isCovered = isOverlapped(document.querySelector('#'+vm.id));
    	        	
    	        	if(visible && !isCovered) {
    	        		vm.refresh();
    	        	}else {
    	        		vm.timerRefresh();
    	        	}
    	        }
    	      }, vm.time * 1000);
    	  }
      },
      indexMethod(index){
    	  if (this.pagination) {
    		  var page = this.$refs.pg.internalCurrentPage,
              rows = this.$refs.pg.internalPageSize;
          
    		  return (page-1)*rows + index + 1;
    	  }else{
    		  return index + 1;
    	  }
      },
      getChecked() {
          return this.ckList;
      },
      getData() { /* 获取已选择的结果数据 */
          return this.tbData;
      },
      sizeChange(size){
        var vm = this;
        vm.pageNum = size;
        vm.frontReload(size);
      },
      currentChage(page){
        var vm = this;
        vm.pageIndex = page;
        vm.frontReload(page);
      },
      frontReload(ev){
        if(this.frontPagination == true) return;
        this.isLoading = true;
        this.refresh(ev);
      },
      reloadTb(ev) {
        this.isLoading = true;
        this.refresh(ev);
      },
      /* table APIs */
      clearSelection() {
    	  this.ckList = [];
        this.$refs.ctableInner.clearSelection();
      },
      toggleRowSelection(row, selected) {
        this.$refs.ctableInner.toggleRowSelection(row, selected);
      },
      toggleAllSelection() {
        this.$refs.ctableInner.toggleAllSelection()
      },
      toggleRowExpansion(row, expanded) {
        this.$refs.ctableInner.toggleRowExpansion(row, expanded)
      },
      setCurrentRow(row) {
        this.$refs.ctableInner.setCurrentRow(row)
      },
      clearSort() {
        this.$refs.ctableInner.clearSort()
      },
      clearFilter(columnKey) {
        this.$refs.ctableInner.clearFilter(columnKey)
      },
      doLayout() {
        this.$refs.ctableInner.doLayout()
      },
      sort(prop, order) {
        this.$refs.ctableInner.sort(prop, order)
      },
      appendRow(row) {
        this.rows.push(row);
      },
      deleteRow(keyValue) {
        var vm = this;
        this.rows = this.rows.filter(function(item) {
          return item[vm.tbRowKey] != keyValue;
        });
      },
      updateChecked(key, row) {
        var vm = this;
        if (key) {
          setTimeout(function() {
            var value = row[key];
            if (vm.ckList.includes(value)) {
              var index = vm.ckList.indexOf(value)
              vm.ckList.splice(index, 1);
              vm.$refs[value][0].model = false; // update card unchecked
            } else {
              vm.ckList.push(value);
              vm.$refs[value][0].model = true; // update card checked
            }
          }, 0);
        }
      },
      /* table events */
      select(selection, row) {
        this.$emit('select', selection, row);
        this.updateChecked(this.tbRowKey, row);
      },
      selectAll(selection) {
        var vm = this,
            key = vm.tbRowKey;
        vm.$emit('select-all', selection);
        if (key) {
          if (selection.length) {
            var ocks = vm.ckList.map(function(item){return item;})||[];
            selection.map(function(row) {
                if (!vm.ckList.includes(row[key])) {
                  vm.ckList.push(row[key]);
                  vm.$refs[row[key]][0].model = true; // update card checked
                }
            });

            // 检测selection
            const cols = vm.$refs.ctableInner.columns;
            const col = cols.filter(function(item){
              return item.reserveSelection == true && item.type == 'selection';
            })[0];
            if(col) {
              vm.ckList = selection.map(function(row) {
                return row[key];
              });
            }
            
            var pureRows = selection.filter(function(row){ return !ocks.includes(row[key]);});
            if(vm.limit && ocks.length + pureRows.length > vm.limit) {
              var dis = ocks.length + pureRows.length - vm.limit,
                  delRows = pureRows.slice(pureRows.length-dis);

              delRows.map(function(row){
                vm.toggleRowSelection(row,false);
                var index = vm.ckList.indexOf(row[key]);
                if (index >= 0) {
                  vm.ckList.splice(index, 1);
                }
              })
            }
            
          } else {
            vm.tbData.map(function(row) {
              var index = vm.ckList.indexOf(row[key]);
              if (index >= 0) {
                vm.ckList.splice(index, 1);
                vm.$refs[row[key]][0].model = false; // update card unchecked
              }
            });
          }
        }
      },
      selectionChange(selection) {
        this.$emit('selection-change', selection,this);
        
        var vm = this;
        /*
        if(vm.isLoaded && selection && selection.length == 0 && vm.rowKey) {// 全取消
          // frontData  rowKey
          var vm = this,
              dKeys = vm.frontData.map(function(row){
                return row[vm.rowKey]
              });

          dKeys.map(function(key){
            if(vm.ckList.includes(key)) {
              var index = vm.ckList.indexOf(key);
              if(index >= 0) {
                vm.ckList.splice(index,1);
              }
            }
          })
        }
        if(vm.isLoaded && selection && selection.length > 0 && vm.rowKey) {// 全取消
          var vm = this,
              dKeys = vm.frontData.map(function(row){
                return row[vm.rowKey]
              }),
              sKeys = selection.map(function(row){
                return row[vm.rowKey]
              });

          dKeys.map(function(key){
            if(!sKeys.includes(key) && vm.ckList.includes(key)) {
              var index = vm.ckList.indexOf(key);
              if(index >= 0) {
                vm.ckList.splice(index,1);
              }
            }
          })
        }
        */
        if(vm.limit){
          vm.$nextTick(function(){
            setTimeout(function(){
              var tb = vm.$refs.tbctn,
                  isLess = vm.ckList.length<vm.limit,
                  uncheckes = tb.querySelectorAll('label.el-checkbox');
              
              if(isLess){ // 未到达最大限制时
                tb.classList.remove('limit');
              }else {
                tb.classList.add('limit');
              }
              
              Array.from(uncheckes).map(function(item){
                if(isLess) {
                  item.removeEventListener('click',forbiddenEvent,true);
                }else {
                  item.addEventListener('click',forbiddenEvent,true);
                }
              })
            },50);
          });
        }
      },
      cellMouseEnter(row, column, cell, event) {
        this.$emit('cell-mouse-enter', row, column, cell, event);
      },
      cellMouseLeave(row, column, cell, event) {
        this.$emit('cell-mouse-leave', row, column, cell, event);
      },
      cellClick(row, column, cell, event) {
        this.$emit('cell-click', row, column, cell, event);
      },
      cellDblclick(row, event) {
        this.$emit('cell-dblclick', row, event);
      },
      rowClick(row, event, column) {
        this.$emit('row-click', row, event, column);
      },
      rowContextmenu(row, event) {
        this.$emit('row-contextmenu', row, event);
      },
      rowDblclick(row, event) {
        this.$emit('row-dblclick', row, event);
      },
      headerClick(column, event) {
        this.$emit('header-click', column, event);
      },
      fitColumn(column){
      	var nodes = document.querySelectorAll('.'+column.id),
      		fitWidth = '';
      	if(nodes.length){
      		Array.from(nodes).map(function(node){
      			var $dom = $('<span style="position: absolute;z-index: -10;visibility: hidden;"></span>').text(node.innerText);
      			$('body').append($dom);
      			var domWidth = $dom.width();
      			
      			if(domWidth>fitWidth) fitWidth = domWidth;
      			$dom.remove();
      		});
      		if(fitWidth) fitWidth += 30;
      	}
      	column.width = fitWidth;
        this.doLayout();
      },
      headerContextmenu(column, event) {
      	this.fitColumn(column);
        this.$emit('header-contextmenu', column, event);
    	  event.returnValue = false;
      },
      sortChange({
        column, prop, order
      }) {
        this.sortName = prop;
        this.sortOrder = (order || '').replace('ending', '');
        this.sortChanges = this.sortName + this.sortOrder;

        this.$emit('sort-change', {
          column, prop, order
        });
      },
      filterChange(filters) {
        this.$emit('filter-change', filters);
      },
      currentChange(currentRow, oldCurrentRow) {
        this.$emit('current-change', currentRow, oldCurrentRow,this);
      },
      headerDragend(newWidth, oldWidth, column, event) {
        this.$emit('header-dragend', newWidth, oldWidth, column, event);
      },
      expandChange(row, expandedRows) {
        this.$emit('expand-change', row, expandedRows);
      }
  },
  mounted() {
    var vm = this;
    vm.refresh();
    if (vm.time) {
      // timer存入到window对象上以 vm_tb开头，确保定时器唯一
      /*
      var timerId = 'vm_tb_'+vm.id;
      if (window[timerId]) clearInterval(window['vm_tb_'+vm.id]);
      
      window[timerId] = setInterval(function() {
        var hasDom = false;
        
        if (vm.$refs.tbctn && vm.$refs.tbctn.parentNode) hasDom = true;
        
        if(vm.id) {
    		if( document.querySelector('#'+vm.id) ) {
    			
    		}else {
    			hasDom = false;
    		}
    	}

        if (vm._isDestroyed || !hasDom) clearInterval(window[timerId]);
        else {
        	var visible = isVisible(document.querySelector('#'+vm.id)),
        		isCovered = isOverlapped(document.querySelector('#'+vm.id));
        	
        	if(visible && !isCovered) {
        		vm.refresh();
        	}
        }
      }, vm.time * 1000);
      */
    }
    
    !vm.showPager && vm.$refs.pg.$el.querySelector('.el-pagination__jump').setAttribute('pagenum',vm.pagenum);
  },
  created() {
    var vm = this;
    if (vm.tbRowKey) {
      vm.defaultChecked.map(function(item) {
        vm.ckList.push(item);
      })
    }
  }
});

Vue.component('el-pairgrid', {
  template: [
    '<div :style="gridStyle" class="el-pairgrid">',
      '<div :style="gridStyleLeft">',
        '<div class="el-pairgrid-title">{{title[0]}}</div>',
        '<div class="pairgrid-left">',
          '<div><slot name="prev"></slot></div>',
          '<el-ctable ref="leftTb" :data="leftData" style="overflow: auto;flex: auto;" :rownumber="rownumber" :class="{readonly: !operatable}" :url="pairgridLeftUrl" height="100%" :id="pairgridLeftId" pagination="true" :row-key="rowKey" :query-params="leftQueryParams"',
          ':limit="limit" @load-success="leftLoadSuccess" @select="select" @selection-change="selectChange" @select-all="selectAll">',
            '<slot name="left"></slot>',
            '<slot name="toolbar" slot="toolbar"></slot>',
          '</el-ctable>',
        '</div>',
        '<div v-if="!isNormal" class="el-pairgrid-title" style="position: absolute;right: 0px;display: flex;align-items: center;">',
          '{{title[1]}} (<span style="color:4d84ff;">{{checkedList.length}}</span>) &nbsp;<i @click="rightGridShow = !rightGridShow" :class="countBtStyle" style="font-size: 12px;"></i>',
        '</div>',
      '</div>',
      '<div v-if="isNormal" style="padding: 5px"></div>',
      '<div :style="gridStyleRight" :class="pairgridRight">',
        '<div v-if="isNormal" class="el-pairgrid-title">{{title[1]}}</div>',
        '<el-ctable ref="rightTb" :url="pairgridRightUrl" height="100%" :rownumber="rownumber" :filterstr="filterStr" :match-keys="matchKeys" :id="pairgridRightId" @load-success="rightLoadSuccess" pagination="true" :front-pagination="true" :row-key="rowKey">',
          '<slot name="right"></slot>',
          '<el-table-column width="1" v-if="operatable">',
          '<template slot-scope="scope">',
            '<div slot="reference" class="opts-wrapper" style="border-radius: 18px;">',
            '<i @click="deleteChecked(scope.row)" class="el-icon el-icon-circle-close" style="font-size:16px;"> </i>',
            '</div>',
          '</template>',
          '</el-table-column>',
          '<template slot="toolbar">',
            '<div style="display: flex;">',
              '<div class="rightQueryBoxCls" style="flex: 1 auto;border:1px solid #DEDFE6;border-radius:2px;max-width:300px;">',
                '<el-input v-model="searchText" class="pairgrid-query" @keyup.enter.native="query" :placeholder="msges.placeholder" :value="searchText" size="small" style="padding-left:20px;"></el-input>',
                '<i @click="query" class="el-icon el-icon-common-search"></i>',
              '</div>',
              '<div v-if="operatable" style="padding-left: 15px;">',
                '<div @click="deleteAll" class="clear-all el-icon el-icon-operation-delete"></div>',
              '</div>',
            '</div>',
          '</template>',
        '</el-ctable>',
        '<div v-if="countShow" style="display: none;background: #fff;font-size: 13px;height: 20px;padding: 0px 20px 9px 20px;text-align: right;border: 1px solid #D1ECF5;border-top-width: 0px;">{{totalStr}} {{selectedCount}} {{unitStr}}</div>',
      '</div>',
    '</div>'
  ].join(' '),
  props: {
    title: {
      type: Array,
      default: ['', '']
    },
    rightUrl: String,
    leftUrl: String,
    rowKey: String,
    height: [String, Number],
    queryName: String,
    zone: {
      type: Array,
      default: [60, 40]
    },
    messages: {
      type: Object,
      default: function() {
        return {}
      }
    },
    queryParams: {
      type: Object,
      default: function() {
        return {}
      }
    },
    id: {
        type: String,
        default: ''
    },
    readonly: {
      type: Boolean,
        default: false
    },
    rownumber: {
      type: Boolean,
        default: false
    },
    model: String,
    limit: {
    	type: Number,
    	default: ''
    },
    gridData: {
      type: Object,
      default: function() {
        return {}
      }
    }
  },
  data() {
    return {
      checkedList: [],
      originList: [],
      searchText: '',
      filterStr: '',
      rightGridShow: false
    }
  },
  computed: {
      dataList() {
        return this.gridData;
      },
      leftData() {
        var vm = this;

        if(vm.gridData) {
          return vm.gridData.left;
        }else {
          return [];
        }
      },
      rightData() {
        var vm = this;

        if(vm.gridData) {
          return vm.gridData.right;
        }else {
          return [];
        }
      },
      leftQueryParams(){
        return this.queryParams? this.queryParams: {};
      },
      operatable(){
        return this.readonly === false;
      },
      pairgridRightUrl(){
        return this.rightUrl;
      },
      pairgridLeftUrl(){
        return this.leftUrl;
      },
      pairgridLeftId(){
        return this.id+'_left';
      },
      pairgridRightId(){
        return this.id+'_right';
      },
      countShow(){
        return true;
      },
      totalStr(){
        if(isLocalZH) return '共';
        else return 'Total';
      },
      unitStr(){
        if(isLocalZH) return '条';
        else return '';
      },
      selectedCount(){
        return this.checkedList.length;
      },
      gridStyle() { /* 容器的样式 */
        var style = {
          display: 'flex',
          position: 'relative',
          'min-height': '300px'
        };
        if (this.height) style.height = this.height;
        return style;
      },
      gridStyleLeft() { /* 左表容器的样式 */
        return {
          flex: '1 1 ' + this.zone[0] + '%',
          overflow: 'hidden',
          display: 'flex',
          'flex-direction': 'column'
        }
      },
      isNormal(){
    	  return this.model == 'normal';  
      },
      pairgridRight(){
        return {
          'pairgrid-right': true,
          'is-normal': this.model == 'normal'
        }  
      },
      gridStyleRight() { /* 右表容器的样式 */
        var style = {
                  flex: '1 1 ' + this.zone[1] + '%',
                  overflow: 'auto',
                  display: 'flex',
                  'flex-direction': 'column',
          };
        if(!this.isNormal){
          Object.assign(style,{
                  position: 'absolute',
                  height: 'calc(100% - 30px)',
                  top: '28px',
                  right: 0,
                  visibility: this.rightGridShow?'visible': 'hidden',
                  'box-shadow': '1px 2px 8px #e3e3e3',
                  'z-index': 100,
                  'min-width': '40%',
                  'max-width': '200px'
          })
        }
        return style;
      },
      countBtStyle(){
        return {
          'el-icon': true,
          'el-icon-circle-down': !this.rightGridShow,
          'el-icon-circle-up': this.rightGridShow
        }
      },
      matchKeys() {
        return this.queryName ? this.queryName.split(',') : [this.rowKey]
      },
      msges() { /* 涉及到国际的信息 */
        return Object.assign({
          tips: QueRen,
          delete: QueDingQingKongYiXuanShuJu,
          placeholder: this.rowKey,
          okText: QueDing,
          cancelText: QuXiao
        }, this.messages);
      }
  },
  watch: {
    dataList: {
      handler: function(obj) {
        var vm = this,
            oTb = vm.$refs.leftTb,
            rowKey = vm.rowKey,
            rows = obj.right;
        
        vm.$nextTick(function(){
            // 回显勾选逻辑: 需要兼容多选、单选模式
            if(rowKey && rows) {
              var selection = oTb.$refs.ctableInner.store.states.selection,
                skeys = selection.map(function(item){ return item[rowKey]}),
                tRows = oTb.getData();
              
              // 回显选中，似乎store中预先有数据时，渲染时才会自动选中匹配项
              rows.map(function(row) {
                var srow = row;
                if(tRows.length) {
                  var frow = tRows.filter(function(item){
                      return item[rowKey] == row[rowKey]
                    })[0];
                  
                  frow && (srow = frow);

                  if(!vm.$refs.leftTb.ckList.includes(srow[rowKey])) {
                    selection.push(srow);
                    oTb.select([srow], srow);
                  }
                }else {
                  if(!vm.$refs.leftTb.ckList.includes(srow[rowKey])) {
                    oTb.toggleRowSelection(srow, true);
                  }
                }
              });
            }
        })
      },
      deep: true
    }
  },
  methods: {
      appendCheckedRows(rows) {
        var vm = this,
            data = {
              total: rows.length,
              rows: rows
            };

        rows.map(function(row){
        	if (!vm.checkedList.includes(row[vm.rowKey])) {
        		 vm.$refs.rightTb.rows.push(row);
                 vm.checkedList.push(row[vm.rowKey]);
                 var ckList = vm.$refs.leftTb.ckList;
                 if(!ckList.includes(row[vm.rowKey])) {
                    ckList.push(row[vm.rowKey]);
                 }
            }
        });
        vm.$refs.leftTb.refresh();
        vm.$emit('checked-change');
        vm.$emit('right-load-success', rows);
        vm.$emit('selection-change', rows);
      },
      appendCheckedRow(row) {
        var vm = this,
            rRows = vm.$refs.rightTb.rows;
        
        if (vm.checkedList.includes(row[vm.rowKey])) {
          vm.$message({
            message: 'Existed!',
            type: 'warning'
          });

          return;
        }

        vm.checkedList.push(row[vm.rowKey]);
        rRows.push(row);
        vm.$emit('checked-change');
        vm.$emit('selection-change', rRows);
      },
      updateCheckeds(vm, row, forbidden) { /* 更新右表的选中结果 */
        var rRows = vm.$refs.rightTb.rows;
        var rcklist = vm.$refs.rightTb.ckList;

        if (vm.checkedList.includes(row[vm.rowKey])) {
          vm.$refs.rightTb.rows = rRows.filter(function(item) {
            return item[vm.rowKey] != row[vm.rowKey]
          });
          vm.checkedList = vm.checkedList.filter(function(item) {
            return item != row[vm.rowKey]
          });
          var idx = rcklist.indexOf(row[vm.rowKey]);
          if (idx >= 0) {
            rcklist.splice(idx, 1);
          }
        } else {
          vm.checkedList.push(row[vm.rowKey]);
          rRows.push(row);
          rcklist.push(row[vm.rowKey]);
        }
        !forbidden && vm.$emit('checked-change');
      },
      selectChange(selection) {
        this.$emit('selection-change', selection, this);
      },
      select(selection, row) {
        this.updateCheckeds(this, row);
      },
      selectAll(selection) {
        var vm = this,
            lRows = vm.$refs.leftTb.rows;
        if (selection.length) {
          var ocks = vm.$refs.rightTb.ckList.map(function(item){return item;})||[];
          lRows.map(function(row) {
            if (!vm.checkedList.includes(row[vm.rowKey])) vm.updateCheckeds(vm, row, true);
          })

          var pureRows = selection.filter(function(row){ return !ocks.includes(row[vm.rowKey]);});
          if(vm.limit && ocks.length + pureRows.length > vm.limit) {
            var dis = ocks.length + pureRows.length - vm.limit,
                delRows = pureRows.slice(pureRows.length-dis);

            delRows.map(function(row){
              var idx = -1;
              vm.$refs.rightTb.rows.map(function(item,i){
                if(item[vm.rowKey]==row[vm.rowKey]) idx = i;
              });
              if (idx >= 0) {
                vm.$refs.rightTb.rows.splice(idx, 1);
              }

              var index = vm.$refs.rightTb.ckList.indexOf(row[vm.rowKey]);
              if (index >= 0) {
                vm.$refs.rightTb.ckList.splice(index, 1);
              }

              index = vm.checkedList.indexOf(row[vm.rowKey]);
              if (index >= 0) {
                vm.checkedList.splice(index, 1);
              }
            });

            vm.$nextTick(function(){
              vm.$refs.leftTb.$refs.ctableInner.store.states.isAllSelected = false;
            });
          }
        } else {
          lRows.map(function(row) {
            vm.updateCheckeds(vm, row, true);
          })
        }
        vm.$emit('checked-change');
      },
      rightLoadSuccess(data) {
        var vm = this;
        vm.$nextTick(function(){
	        if (data && data.rows) {
	          vm.originList = data.rows.map(function(item) {
	            return item[vm.rowKey];
	          });
	          vm.checkedList = data.rows.map(function(item) {
	            return item[vm.rowKey];
	          });

            vm.$refs.leftTb.clearSelection();
	          var lRows = vm.$refs.leftTb.rows;
	          lRows.map(function(row) {
	            if (vm.checkedList.includes(row[vm.rowKey])) {
	              vm.$refs.leftTb.toggleRowSelection(row, true)
	            }
	          });
            vm.$refs.leftTb.ckList = vm.checkedList.map(function(item){ return item;});
            vm.$refs.rightTb.ckList = vm.checkedList.map(function(item){ return item;});
	        }
	        vm.$emit('right-load-success', data?data.rows:[]);
          vm.$emit('selection-change', data?data.rows:[]);
        });
      },
      leftLoadSuccess(data) {
        var vm = this;
          vm.$nextTick(function(){
              if (vm.checkedList.length) {
	        	  var lRows = vm.$refs.leftTb.rows
	              lRows.map(function(row) {
	                if (vm.checkedList.includes(row[vm.rowKey])) {
	                  vm.$refs.leftTb.toggleRowSelection(row, true);
	                }
	              });
              }
          });
      },
      deleteChecked(row,forbidden) { /* 右表单行删除 */
        var vm = this,
          lRows = vm.$refs.leftTb.rows;
        this.updateCheckeds(vm, row, forbidden);
        var ckList = vm.$refs.leftTb.ckList;
        lRows.map(function(item) {
          if (row[vm.rowKey] == item[vm.rowKey]) {
        	  vm.$refs.leftTb.toggleRowSelection(item, false);
          }
        });
        var index = ckList.indexOf(row[vm.rowKey]);
        if(index >=0) {
      	  ckList.splice(index,1);
        }

        // 
        var rightckList = vm.$refs.rightTb.ckList;
        var idx = rightckList.indexOf(row[vm.rowKey]);
        if(idx >=0) {
      	  rightckList.splice(idx,1);
        }
      },
      deleteAll() { /* 右表批量删除 */
        var vm = this,
          rRows = vm.$refs.rightTb.rows;
        if (rRows.length) {
          this.$confirm(this.msges.delete, this.msges.tips, {
            confirmButtonText: this.msges.okText,
            cancelButtonText: this.msges.cancelText,
            type: 'warning'
          }).then(() => {
            /*
            rRows.map(function(row) {
              vm.deleteChecked(row,true);
            });
            */
            vm.$refs.leftTb.clearSelection();
            vm.$refs.rightTb.clearSelection();
            vm.$refs.rightTb.rows = [];
            vm.$refs.leftTb.ckList = [];
            vm.$refs.rightTb.ckList = [];
            vm.checkedList = [];

            vm.$emit('checked-change');
          }).catch(() => {

          });
        }
      },
      clear() { /* 右表批量删除 */
          var vm = this,
              rRows = vm.$refs.rightTb.rows;
          if (rRows.length) {
	          rRows.map(function(row) {
	            vm.deleteChecked(row);
	          });
          }
          vm.$refs.leftTb.ckList = [];
      },
      query(ev) {
        var vm = this,
          stxt = vm.searchText.trim(),
          rRows = vm.$refs.rightTb.rows,
          fields = vm.matchKeys,
          ctn = null,
          node = ev.target,
          notFind = true;
        // 新查询逻辑
        vm.filterStr = stxt;

        // 历史查询功能暂不执行
        return;
        while (notFind) { /* 迭代查找右表容器 */
          if (Array.from(node.classList).includes('pairgrid-right')) {
            ctn = node
            notFind = false;
          } else {
            node = node.parentNode;
          }
        }
        
        rRows.map(function(row, idx) {
          var isMatch = false;
          if (stxt) {
            for (var key in row) {
              if (row.hasOwnProperty(key) && fields.includes(key)) {
                if ((row[key]+''||'').indexOf(stxt) >= 0) isMatch = true;
              }
            }
          } else isMatch = true;

          var tr = ctn.querySelectorAll(
            '.el-table__body-wrapper table tbody tr')[idx];
          if (!isMatch) {
            tr.classList.add('gridHidden');
          } else {
            tr.classList.remove('gridHidden');
          }
        });
      },
      getData() { /* 获取已选择的结果数据 */
        return this.$refs.rightTb.rows;
      },
      hasChanged() {
        var cStr = this.checkedList.sort().join(' '),
          oStr = originList.sort().join(' ');
        return cStr == oStr
      },
      reload(ev) {
        var vm = this;
        setTimeout(function(){
          if(vm.$refs.leftTb) vm.$refs.leftTb.reloadTb(ev)
        },100);
      },
      reloadRightTb(){
    	  this.$refs.rightTb.refresh();
      }
  }
});

Vue.directive('clickoutside', {
  bind: function(el, binding, vode) {
    function documentHandler(e) {
      if (el.contains(e.target)) {
        return false
      }
      if (binding.expression) {
        binding.value(e)
      }
    }
    el.__vueClickOutSide__ = documentHandler
    document.addEventListener('click', documentHandler)
  },
  unbind: function(el, binding) {
    document.removeEventListener('click', el.__vueClickOutSide__)
    delete el.__vueClickOutSide__
  }
})

Vue.component('el-query', {
  template: [
    '<div class="el-query" style="position: relative;" v-clickoutside="handerClose">',
    '<div class="advanceQuery" style="width:fit-content;position:relative">',
    '<el-input @focus="handerFocus" @blur="handerBlur" @keyup.enter.native="query" v-model="searchText" class="pairgrid-query" :placeholder="placeholder" :value="searchText" size="small"></el-input>',
      '<i v-if="arrowShow" :class="arrowClass" @click="arrowClick"></i>',
    '<i @click="query" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>', 
      '<label v-if="arrowShow" class="arrow-text mainCol" :style="arrowTxtStyle" @click="arrowClick">{{arrowText}}</label>',
    '</div>',
    '<transition name="el-zoom-in-top">',
    '<div v-show="queryShow" class="transition-box">',
    '<slot name="form"></slot>',
    '<div style="padding-top: 20px;">',
    '<el-button type="primary" size="mini" @click="advanceQuery">{{okText}}</el-button>',
    '<el-button size="mini" @click="resetQuery">{{resetText}}</el-button>',
    '</div>',
    '</div>',
    '</transition>',
    '</div>'
  ].join(' '),
  props: {
    placeholder: String,
    okText: {
      type: String,
      default: vueComponetsMsg.query
    },
    resetText: {
      type: String,
      default: vueComponetsMsg.reset
    },
    arrowText: {
      type: String,
    	default: vueComponetsMsg.advanceQuery
    },
    type: {
        type: String,
        default: ''
    }
  },
  data() {
    return {
      searchText: '',
      queryShow: false,
      arrowTxtShow: false
    }
  },
  computed: {
    arrowClass() {
      return {
        'el-icon': true,
        'el-icon-common-query-down': !this.queryShow,
        'el-icon-common-query-up': this.queryShow
      }
    },
    arrowTxtStyle(){
       return {
	      visibility: this.arrowTxtShow? 'visible':'hidden'
	   }
      },
      arrowShow() {
        return this.type != 'normal';
    }
  },
  methods: {
    query(ev) {
        this.$emit('query', this.searchText, ev);
        this.handerClose()
      },
      advanceQuery(ev) {
      	this.searchText = '';
        this.$emit('advance-query', ev);
        this.handerClose()
      },
      resetQuery(ev) {
        this.$emit('reset', ev);
      },
      arrowClick(ev) {
        this.queryShow = !this.queryShow;
        if(this.queryShow){
           this.$emit('open-advance', ev);
        }
      },
      handerClose(ev) {
        this.queryShow = false;
          this.arrowTxtShow = false;
      },
      handerFocus(ev){
    	  this.handerClose(ev);
          this.arrowTxtShow = true;
      },
      handerBlur(ev) {
        this.arrowTxtShow = false;
      },
      reset() {
    	  this.searchText = '';
      }
  }
});

Vue.component('el-slide', {
  template: [
    '<transition :name="animateName">',
    '<div v-show="show" :style="slideSytel" :class="elSlideClass">',
    '<div v-if="showMask" class="el-dialog__wrapper" style="z-index: -1;background-color: #000000;opacity:0.3;"></div>',
    '<el-card class="box-card" :footer="footer" :body-style="bodyStyle" style="display: flex;flex-direction: column;flex: auto;">',
    '<div v-if="showHeader" slot="header" class="clearfix">',
    '<span>{{title}}<slot name="subTitle"></slot></span>',
    '<span style="position: absolute;right: 10px;top: 10px;display: flex;align-items: center;" >',
    '<slot name="toolbar"></slot>',
    '<span @click="handerCancel"><i class="el-icon-close el-icon"></i></span>',
    '</span>',
    '</div>',
    '<div v-loading="loading" style="flex: 1 auto;overflow: auto;position: relative;">',
    '<div ref="htmlSegment" v-show="hasUrl && !loading" class="slide-content" style="height: 100%;"></div>',
    '<slot v-if="!hasUrl"></slot>',
    '</div>',
    '<div slot="footer" style="padding-top:10px;">',
    '<el-button-group size="mini">',
    '<el-button type="primary" size="mini" @click="handerOk" :loading="showSubmitLoading">{{okText}}</el-button>',
    '<el-button size="mini" @click="handerCancel">{{cancelText}}</el-button>',
    '</el-button-group>',
    '</div>',
    '</el-card>',
    '</div>',
    '</transition>'
  ].join(' '),
  props: {
    title: {
      type: String,
      default: vueComponetsMsg.title
    },
    okText: {
      type: String,
      default: vueComponetsMsg.ok
    },
    cancelText: {
      type: String,
      default: vueComponetsMsg.cancel
    },
    url: String,
    show: {
      type: Boolean,
      default: false
    },
    position: {
      type: String,
      default: 'left'
    },
    forcePosition: {
      type: Boolean,
      default: false
    },
    height: {
      type: [String, Number],
      default: '100%'
    },
    width: {
      type: [String, Number],
      default: '100%'
    },
    footer: {
      type: Boolean,
      default: true
    },
    modal: {
      type: Boolean,
      default: true
    },
    header: {
        type: Boolean,
        default: true
    },
    subloading: {
        type: Boolean,
        default: false
    },
    method: {
      type: String,
      default: 'post'
    }
  },
  data() {
    return {
      htmlstr: '',
      loading: true,
      params: {},
      cb:''
    }
  },
  computed: {
	  elSlideClass(){
		  return {
			  'el-slide': true,
			  'slide-position-bottom': this.position == 'bottom',
			  'slide-position-top': ['top','left','right'].includes(this.position)
		  } 
	  },
	  showHeader(){
		  return this.header ? true : false;
	  },
	  showMask() { /* 控制是否有遮蔽层 */
		if(this.forcePosition) {
			if(this.position == 'left' || this.position == 'right') return true;
			else return false;
		}else {
	        return false;
		}
        // if(this.position == 'left' || this.position == 'right') return true;
        // else return false;
        //return this.modal == true || this.modal == 'true';
      },
      hasUrl() {
        return this.url ? true : false;
      },
      animateName() { /* 滑出方式的映射 */
        var codes = {
          top: 'el-zoom-in-top',
          bottom: 'el-zoom-in-bottom',
          center: 'el-zoom-in-center',
          left: 'el-zoom-in-top'||'left',
          right: 'el-zoom-in-top'||'left'
        }
        if(this.forcePosition) {
          Object.assign(codes,{left: 'left', right: 'left'});
        }
        
        return codes[this.position]
      },
      slideSytel() { /* 滑出层根据方位调整布局 */
        var style = {
          width: '100%',
          height: this.height,
          display: 'flex',
          position: 'absolute',
          'background-color': '#fff',
          'z-index': '2000'
        };
        if (!isNaN(this.height)) {
          style.height += 'px'
        }
        if (this.position == 'left' || this.position == 'right') {
          style.width = '100%';
          style.right = '0px';
          // if (this.width) {
          //   style.width = typeof this.width == 'string' ? this.width : this
          //     .width + 'px';
          // }
        }
        if (this.position == 'bottom') {
          style.bottom = 0;
          style.left = 0;
        } else {
          style.top = 0
        }
        if(this.forcePosition) {
          if (this.width) {
            style.width = typeof this.width == 'string' ? this.width : this.width + 'px';
          }
        }

        return style;
      },
      bodyStyle() {
        return {
          display: 'flex',
          'flex-direction': 'column',
          flex: '1 auto',
          height: '100%',
          overflow: 'auto'
        }
      },
      url_show() {
        return this.url + this.show;
      },
      showSubmitLoading(){
        return this.subloading ? true : false;
      }
  },
  watch: {
    url_show() {
      if (this.show) this.getHtml();
    }
  },
  methods: {
    getHtml() { /* 通过url获取远程页面 */
        var vm = this,
          queryData = {};
        if (this.params) Object.assign(queryData, this.params);

        if (vm.url) {
          vm.loading = true;
          vm.$refs.htmlSegment.innerHTML = '';
          var method = vm.method;
          $.ajax({
            type: method,
            url: vm.url,
            data: queryData,
            dataType: 'html',
            success: function(html) {
              vm.$refs.htmlSegment.innerHTML = html;
              var scripts = vm.$refs.htmlSegment.querySelectorAll('script');
              setTimeout(function() {
                Array.from(scripts).map(function(script) { /* 执行远程的脚本 */
                  if (vm.$refs.htmlSegment.contains(script)) {
                    vm.$refs.htmlSegment.removeChild(script);
                  }
                  var newScript = document.createElement(
                    'script');
                  newScript.type = 'text/javascript';
                  newScript.innerHTML = script.innerHTML;
                  vm.$refs.htmlSegment.appendChild(newScript);
                });
                if(vm.cb && typeof vm.cb == 'function')vm.cb();
              }, 0);
              
              setTimeout(function() {
                  vm.loading = false;
                  try{
                	  $.parser.parse(vm.$refs.htmlSegment);
                  }catch(e){}
              }, 0);
            },
            error: function() {
              vm.htmlstr = 'error...';
              vm.loading = false;
              if(vm.cb && typeof vm.cb == 'function') vm.cb();
            }
          });
        } else {
          vm.loading = false;
          if(vm.cb && typeof vm.cb == 'function') vm.cb();
        }
      },
      clear() {
        var vm = this;

        if(vm.url) {
          vm.$refs.htmlSegment.innerHTML = '';
        }
      },
      hide() {
        var vm = this;
        vm.show = false;
        vm.modal = false;
        vm.$refs.htmlSegment.innerHTML = "";
      },
      showSlide(params,cb) {
    	  var vm = this;
        vm.show = true;
        if(typeof params == 'function'){
        	vm.cb = params;
        	vm.params = '';
        }else {
        	vm.params = params;
        	vm.cb = cb;
        }
        
        vm.$nextTick(function(){
            if(['left','right'].includes(this.position)) vm.modal = true;
        })
      },
      handerOk(ev) {
        this.$emit('ok', ev)
      },
      handerCancel(ev) {
        this.$emit('cancel', ev);
      }
  }
});

Vue.component('el-cmenu-item', {
  template: [
    '<div :class="cmenuItemCls" @click="itemClick">',
    '<span v-if="!hasChild">{{data.label}}</span>',
    '<div class=" menu-group" v-if="hasChild">',
    '<el-cmenu-item v-for="item in data.child" :data="item" @menu-click="menuClick"></el-cmenu-item>',
    '</div>',
    '</div>'
  ].join(' '),
  props: {
    data: {
      type: Object,
      default: function() {
        return {}
      }
    }
  },
  data() {
    return {

    }
  },
  computed: {
    hasChild() {
        var has = false;
        if (this.data.child && this.data.child.length) has = true;
        return has;
      },
      cmenuItemCls() {
        var cls = {
          'cmenu-item': true,
          'has-child': this.hasChild,
          disabled: this.data.disable,
          'menu-hidden': this.data.show === false ? true : false
        };
        if (this.data.cls) cls[this.data.cls] = true;
        return cls;
      }
  },
  methods: {
    itemClick(ev) {
    	if (this.data.disable != true) {
    		if(this.data.child && this.data.child.length){
    			 ev.stopPropagation();
    		}else this.$emit('menu-click', this.data, ev);
    	}
      },
      menuClick(row, ev) {
        if (row.disable != true) {
          this.$emit('menu-click', row, ev);
          this.$emit('close-menu', row, ev);
        }
       
      }
  }
});

Vue.component('el-cmenu', {
  template: [
    '<transition :name="animateName">',
    '<div ref="menu" v-show="isShow" :class="ctnCls" :style="ctnStyle">',
    '<el-cmenu-item v-for="item in data" :data="item" @menu-click="menuClick" @close-menu="hide"></el-cmenu-item>',
    '</div>',
    '</transition>'
  ].join(' '),
  props: {
    data: {
      type: Array,
      default: []
    },
    align: {
      type: String,
      default: 'left'
    }
  },
  data() {
    return {
      isShow: false,
      top: '',
      left: '',
      animateName: 'el-zoom-in-top'
    }
  },
  computed: {
    ctnCls() {
        return {
          cmenu: true,
          'item-left': this.align == 'left',
          'item-right': this.align == 'right'
        }
      },
      ctnStyle() {
        var style = {};

        if (this.top) style.top = this.top;
        if (this.left) style.left = this.left;
        return style;
      }
  },
  methods: {
    getOffsetTop: function(el) {
      return el.offsetParent ? el.offsetTop + this.getOffsetTop(el.offsetParent) :
        el.offsetTop
    },
    getOffsetLeft: function(el) {
      return el.offsetParent ? el.offsetLeft + this.getOffsetLeft(el.offsetParent) :
        el.offsetLeft
    },
    reposition(target, reference, vm) {
      var docH = window.innerHeight || document.documentElement.clientHeight ||
        document.body.clientHeight,
        tRect = reference.getBoundingClientRect(),
        mRect = target.getBoundingClientRect();
      /**
       * 根据相对视口的坐标偏移，计算相对位置的偏移量
       *（top偏移：rect的top的坐标偏移差，left偏移：rect的left坐标偏移差）
       **/
      var tTop = tRect.top,
        tLeft = tRect.left,
        mTop = mRect.top,
        mLeft = mRect.left,
        top = target.offsetTop + (tTop - mTop) + tRect.height + 12,
        left = target.offsetLeft + (tLeft - mLeft);
      var offsetD = tRect.y + tRect.height + 12;

      if (docH - offsetD < mRect.height) {
        vm.animateName = 'el-zoom-in-bottom';
        top = top - mRect.height - tRect.height - 24;
      }
      return {
        left: left,
        top: top
      };
    },
    show(ev) {
      var vm = this,
        menu = vm.$refs.menu;
      vm.animateName = 'el-zoom-in-top';
      if (ev) {
        menu.style.display = 'block';
        menu.style.opacity = 0;

        var pos = this.reposition(menu, ev.target, vm);
        vm.top = pos.top + 'px';
        vm.left = pos.left + 'px';
        
        ev.stopPropagation();
        setTimeout(function() {
          menu.style.opacity = 1;
        }, 0);
      }
      vm.isShow = true;
    },
    hide() {
      this.isShow = false;
    },
    menuClick(row, ev) {
      this.$emit('click', row, ev);
    }
  }
});
Vue.component('el-topbutton',{
	template:[
		'<div :class="titleButtonClass">',
		'<el-button @mouseover.native="showText" @mouseout.native="hideText" :icon="icon" type="primary" circle @click="clickEvent"></el-button>',
		'<div class="elTopButtonText mainCol" v-show="showFlag">{{title}}</div>',
		'</div>'
	].join(''),
	props:{
		title:{
			type: String,
			default: 'add'
		},
		icon:{
			type: String,
			default: 'el-icon-plus'
		},
		permission:{
			type: String
		}
	},
	data(){
		return{
			showFlag:false
		}
	},
	computed: {
		titleButtonClass() {
			if(this.permission) return "circleIcon " + this.permission;
			else return "circleIcon"
		}
	},
	methods:{
		showText(){
			this.showFlag = true;
		},
		hideText(){
			this.showFlag = false;
		},
		clickEvent(){
			this.$emit('click');
		}
	}
})
Vue.component('el-tslide', {
  template: [
    '<transition :name="animateName">',
    '<div v-show="show" :style="slideSytel" class="el-slide">',
    '<div v-if="showMask" class="el-dialog__wrapper" style="z-index: -1;background-color: #000000;opacity:0.3;"></div>',
    '<el-card class="box-card" :footer="footer" :body-style="bodyStyle" style="display: flex;flex-direction: column;flex: auto;">',
    '<div slot="header" class="clearfix">',
    '<span>{{title}}</span>',
    '<span style="position: absolute;right: 20px;top: 0px;display: flex;align-items: center;" >',
    '<slot name="toolbar"></slot>',
    '<el-button type="text" style="padding: 0px" @click="handerCancel"><i class="el-icon-close"></i></el-button>',
    '</span>',
    '</div>',
    '<div v-loading="loading" class="no-opacity-mask" style="flex: 1 auto;overflow: auto;position: relative;">',
    '<div ref="htmlSegment"></div>',
    '<slot v-if="!hasUrl"></slot>',
    '</div>',
    '<div slot="footer" style="padding: 20px;">',
    '<el-button-group size="mini">',
    '<el-button size="mini" @click="handerOther">{{otherText}}</el-button>',
    '<el-button type="primary" size="mini" @click="handerOk">{{okText}}</el-button>',
    '<el-button size="mini" @click="handerCancel">{{cancelText}}</el-button>',
    '</el-button-group>',
    '</div>',
    '</el-card>',
    '</div>',
    '</transition>'
  ].join(' '),
  props: {
    title: {
      type: String,
      default: vueComponetsMsg.title
    },
    okText: {
      type: String,
      default: vueComponetsMsg.ok
    },
    cancelText: {
      type: String,
      default: vueComponetsMsg.cancel
    },
    otherText: {
      type: String,
      default: vueComponetsMsg.other
    },
    url: String,
    show: {
      type: Boolean,
      default: false
    },
    position: {
      type: String,
      default: 'left'
    },
    height: {
      type: [String, Number],
      default: '100%'
    },
    width: {
      type: [String, Number],
      default: '100%'
    },
    footer: {
      type: Boolean,
      default: true
    },
    modal: {
      type: Boolean,
      default: true
    }
  },
  data() {
    return {
      htmlstr: '',
      loading: true,
      params: {},
      cb:''
    }
  },
  computed: {
    showMask() { /* 控制是否有遮蔽层 */
        return this.modal == true || this.modal == 'true';
      },
      hasUrl() {
        return this.url ? true : false;
      },
      animateName() { /* 滑出方式的映射 */
        var codes = {
          top: 'el-zoom-in-top',
          bottom: 'el-zoom-in-bottom',
          center: 'el-zoom-in-center',
          left: 'left',
          right: 'left'
        }
        return codes[this.position]
      },
      slideSytel() { /* 滑出层根据方位调整布局 */
        var style = {
          width: this.width,
          height: this.height,
          display: 'flex',
          position: 'absolute',
          'background-color': '#fff',
          'z-index': '2000'
        };
        if (!isNaN(this.height)) {
          style.height += 'px'
        }
        return style;
      },
      bodyStyle() {
        return {
          display: 'flex',
          'flex-direction': 'column',
          flex: '1 auto'
        }
      },
      url_show() {
        return this.url + this.show;
      }
  },
  watch: {
    url_show() {
      if (this.show) this.getHtml();
    }
  },
  methods: {
    getHtml() { /* 通过url获取远程页面 */
        var vm = this,
          queryData = {};
        if (this.params) Object.assign(queryData, this.params);

        if (vm.url) {
          vm.loading = true;
          $.ajax({
            type: 'post',
            url: vm.url,
            data: queryData,
            dataType: 'html',
            success: function(html) {
              vm.$refs.htmlSegment.innerHTML = html
              var scripts = vm.$refs.htmlSegment.querySelectorAll('script');
              setTimeout(function() {
                Array.from(scripts).map(function(script) { /* 执行远程的脚本 */
                  if (vm.$refs.htmlSegment.contains(script)) {
                    vm.$refs.htmlSegment.removeChild(script);
                  }
                  var newScript = document.createElement('script');
                  newScript.type = 'text/javascript';
                  newScript.innerHTML = script.innerHTML;
                  vm.$refs.htmlSegment.appendChild(newScript);
                });
                if(vm.cb && typeof vm.cb == 'function') vm.cb();
              }, 0);
              vm.loading = false;
            },
            error: function() {
              vm.htmlstr = 'error...';
              vm.loading = false;
              if(vm.cb && typeof vm.cb == 'function') vm.cb();
            }
          });
        } else {
          vm.loading = false;
          if(vm.cb && typeof vm.cb == 'function') vm.cb();
        }
      },
      hide() {
    	this.show = false;
      },
      showSlide(params,cb) {
        this.show = true;
        if(typeof params == 'function'){
        	this.cb = params;
        	this.params = '';
        }else this.params = params;
      },
      handerOk(ev) {
        this.$emit('ok', ev)
      },
      handerCancel(ev) {
        this.$emit('cancel', ev);
      },
      handerOther(ev){
    	this.$emit('operate',ev);
      }
  }
});
Vue.component('el-add-item', {
	  template: [
		   '<div>',
		   '<el-input v-model="itemVal" :class="itemClass">',
		   '<span @click="handerAdd" slot="suffix" :class="addClass" style="display:inline-block;width:26px;height:26px;top:1px;"></span>',
		   '</el-input>',
		   '<div style="margin-top:5px;">',
		   '<div v-for="item in items" :key="item" class="form-suffix" style="width:164px;line-height:16px;">',
		   '<span class="text" style="width:130px;height:16px;">{{item}}</span>',
		   '<span :code="item" @click="handerDelete" :class="deleteClass"></span>',
		   '</div>',
		   '<div class="el-form-item__error" v-show="errorFlag">{{errorMsg}}</div>',
		   '<div class="el-form-item__error" v-show="existFlag">{{existMsg}}</div>',
		   '</div>',
		   '</div>'
		  ].join(' '),
	  props: {
	    errorMsg: {
	    	type: String,
	    	default: vueComponetsMsg.error
	    },
	    existMsg: {
	    	type: String,
	    	default: vueComponetsMsg.exist
	    },
	    items:{
	    	type:Array,
	    	default:[]
	    },
	    isDisabled:{
	    	type:Boolean,
	    	default:false
	    }
	  },
	  data() {
	    return {
	    	errorFlag:false,
	    	existFlag:false,
	    	itemVal:'',
	    	itemClass:'',
	    	deleteClass:'form-bt-remove operation_delete',
	    	addClass:'titleIcon titleIcon_add inputStar'
	    }
	  },
	  watch:{
		  isDisabled:function(newVal){
			  var vm = this;
			  if(newVal){
				  vm.deleteClass = 'form-bt-remove operation_delete_disabled'
				  vm.addClass = 'titleIcon titleIcon_add_disabled inputStar'
			  }
		  }
	  },
	  mounted(){
		  
	  },
	  methods: {
		  handerAdd(){
			  if(this.isDisabled){
				  return false
			  }else{
				  var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
				  if(this.itemVal != ''&& reg.test(this.itemVal)){
					  if(this.items.includes(this.itemVal)){
						  this.existFlag = true;
						  this.errorFlag = false;
						  this.itemClass='errorItem'
					  }else{
						  this.items.push(this.itemVal)
						  this.itemVal = '';
						  this.existFlag = false;
						  this.errorFlag = false;
						  this.itemClass=''
					  }
				  }else{
					  this.errorFlag = true;
					  this.existFlag = false;
					  this.itemClass='errorItem'
				  } 
			  }
			  
		  },
		  handerDelete(ev){
			if(this.isDisabled){
				return false
			}else{
				var code = ev.target.getAttribute('code')
				this.items.splice(this.items.indexOf(code),1)
			}
		  }
	  }
	});

Vue.component('el-ctree', {
  template: [
  	'<div :style="featureStyle" :class="readonlycls">',
		'<div style="margin-top:10px;margin-bottom:10px;width:440px;" class="queryGroup">',
			'<el-input v-model="filterText" @keyup.enter.native="queryFeature" class="pairgrid-query" :placeholder="placeholder"></el-input>',
			'<i @click="queryFeature" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>',
		'</div>',
		'<div style="width:100%;height:30px;background:#F6F7FB;line-height:30px;display:flex">',
			'<p style="display:inline-block;font-size:14px;padding-left:15px;flex:auto">{{topTitle}}</p>',
			'<div class="tree-op-span" style="padding-top: 5px;">',
				'<!-- Top checkbox -->',
				'<p v-for="(item,index) in checkForms" style="display:inline-block;width:90px;">',
					'<el-checkbox',
            'v-show="!isSpecialMode"',
						'@change="handerMainChange(\'\',\'root\',forms[item.key]);ckboxClick(\'root\',item.key,true);nodeClick()"',
						':indeterminate="indeterminate(\'root\',forms[item.key])"',
						':class="{\'is-checked-cls\': isAllChecked(\'root\',forms[item.key])}"',
						':value="isAllChecked(\'root\',forms[item.key])"',
						':label="true">{{item.label}}</el-checkbox>',
          '<span v-if="isSpecialMode && index == 0" style="font-size: bold;">Aceess</span>',
				'</p>',
			'</div>',
		'</div>',
		'<el-tree ref="featureTree" @node-click="nodeClick" :render-after-expand="false" :default-expanded-keys="defaultExpandedKeys" :data="treeData" node-key="id" accordion :expand-on-click-node="false" :filter-node-method="filterNode" style="height: 100%;overflow: auto;">',
			'<span slot-scope="{node,data}" style="flex: auto;display: flex;">',
				'<span class="treeLabel">{{data.text}}</span>',
				'<span class="tree-op-span">',
					'<!-- Iterative checkboxs -->',
					'<span v-for="(formName,index) in keylist" style="display:inline-block;width:90px;">',
						'<!-- Dir checkbox -->',
						'<el-checkbox',
              'v-show="!isSpecialMode"',
							'v-if="!ignoreNodes[formName].includes(data.id) && data.children && data.children.length && forms[formName].dir[data.id].length"',
							'@change="handerMainChange(\'\',data.id,forms[formName]);ckboxClick(data,formName,true)"',
							'v-model="forms[formName].leaf[data.pid||\'root\']"',
							':indeterminate="indeterminate(data.id,forms[formName])"',
							':class="{\'is-checked-cls\': isAllChecked(data.id,forms[formName]),\'dir-ck-clss\': true}"',
							':cas="updateParent(data.id,data.pid,forms[formName])"',
							':label="data.id" :key="data.id" :name="data.id" :id="data.pid" style="display: flex;">',
							'<span class="el-icon-more-outline"></span>',
						'</el-checkbox>',
						
						'<!-- Leaf checkbox -->',
						'<el-checkbox v-else-if="!ignoreNodes[formName].includes(data.id)" v-model="forms[formName].leaf[data.pid||\'root\']" @change="ckboxClick(data,formName,false)"',
              'v-show="!isSpecialMode"',
							':label="data.id" :key="data.id" :name="data.pid">&nbsp;</el-checkbox>',

            '<span v-if="isSpecialMode && !ignoreNodes[formName].includes(data.id) && (forms[formName].leaf[data.pid||\'root\'].includes(data.id) || forms[formName].leaf[data.id||\'root\'].length)">',
              '{{checkForms[index].readonlyLabel || checkForms[index].label}}</span>',
					'</span>',
				'</span>',
			'</span>',
		'</el-tree>',
	'</div>'
  ].join(' '),
  props: {
    checkForms: {
      type: Array,
      default() {
        return []
      }
    },
    cascade: {
      type: Array,
      default() {
        return []
      }
    },
    data: {
      type: Array,
      default() {
        return []
      }
    },
    ignore: {
      type: Object,
      default() {
        return ''
      }
    },
    readonly: {
      type: Boolean,
      default: false
    },
    readonlyHideCheckbox: {
      type: Boolean,
      default: false
    },
    title: String,
  	defaultExpandedKeys: {
      type: Array,
      default() {
        return []
      }
  	},
    placeholder: String
  },
  data() {
    var vm = this,
        keylist = vm.checkForms.map(function(item) {
          return item.key;
        }),
        ignoreNodes = {},
        base = {
          treeData: [],
          filterText: '',
          keylist: keylist,
          forms: {}
        };
			
    // 设置勾选的响应form
    if(keylist) {
      keylist.map(function(formName){
        base.forms[formName] = {
          dir: {
            root: []
          },
          leaf: {
            root: []
          }
        };
        // 初始化忽略节点
        ignoreNodes[formName] = vm.ignore[formName] || [];
      });

      base.ignoreNodes = ignoreNodes;
    }
    
    return base;
  },
  watch: {
    data(val) {
      this.init();
    },
    checked(val) {
      var vm = this;
      vm.initChecked(vm.treeData);
      vm.reviewForms(vm.treeData);
    }
  },
  computed: {
    topTitle() {
      return this.title || '';
    },
    readonlycls() {
      var vm = this;

      return  {
        'readonly-cls': vm.readonly == true
      }
    },
    featureStyle() {
      return {
        border: '1px solid #DEDFE6',
        'margin-top': '5px',
        display: 'flex',
        'flex-direction': 'column',
        'min-width': '500px'
      }
    },
    isSpecialMode() {
      var vm = this;

      return vm.readonlyHideCheckbox && vm.readonly;
    }
  },
  methods: {
    reset() {
      var vm = this,
          keylist = vm.checkForms.map(function(item) {
            return item.key;
          });

       // 设置勾选的响应form
      if(keylist) {
        keylist.map(function(formName){
          vm.forms[formName] = {
            dir: {
              root: []
            },
            leaf: {
              root: []
            }
          };
        });
      }
    },
    nodeClick(obj,node,t) {
      this.$emit('node-click',obj,node,t);
    },
    //权限树模糊查询
    queryFeature(){
      this.$refs.featureTree.filter(this.filterText);
    },
    resetQuery() {
    	this.filterText = '';
    	this.queryFeature();
    },
    /**
    * 过滤节点方法
    * @param value{string} 当前过滤值
    * @param data{object} 当前节点属性对象
    */
    filterNode(value,data){
      
      var vm = this,
          id = data.id;

      if(vm.isSpecialMode == true){
        var result = vm.getResult(),
          bool = false;

        vm.keylist.map(function(key){
          if(result[key].includes(id)){
            bool = true;
          }
        });

        if(!value) return bool;
        return data.text.indexOf(value) !== -1 && bool;
      }
      
      if(!value) return true;
      return data.text.indexOf(value) !== -1
    },
    /**
     * 初始化勾选状态
     */
    initChecked(list) {
      var vm = this,
          ckMap = vm.checked || {};

      if(list) {
        list.map(function(item){
          if(item.children) vm.initChecked(item.children);
          // 迭代所有check forms
          vm.checkForms.map(function(row){
            var key = item.id;

            ckMap[row.key] && ckMap[row.key].includes(key) && (item[row.prop] = true);
          })
        });
      }
    },
    /* <-------- 核心方法 始 --------> */
    /**
    * 回显权限树勾选项
    * @param list{array} 需要勾选的节点数据
    */
    reviewForms(list){
      var vm = this;
      if(list) {
        list.map(function(item){
          if(item.children) vm.reviewForms(item.children);

          // 迭代所有check forms
          vm.checkForms.map(function(row){
            var status = item[row.prop] == true,
              key = item.id, pkey = item.pid;
            
            !vm.ignoreNodes[row.key].includes(key) && status 
            && vm.forms[row.key].dir[pkey].includes(key) // dir基础数据中含有该节点时，对应clearUpEmpty的净化处理
            && !vm.forms[row.key].leaf[pkey].includes(key) 
            && vm.forms[row.key].leaf[pkey].push(key);
          })
        });
      }
    },
    /**
    * 动态初始化forms为响应式对象
    * @param data{array} 后台返回的数据对象
    * @param form{object} form类型
    */
    reactForms(data,form,formName){
      var vm = this;
      
      if(Array.isArray(data)) {// 数组对象
        data.map(function(item){
          vm.defineKeys(item,form,formName);
        });
      }else{// json对象
        vm.defineKeys(data,form,formName);
      }
    },
    /**
    * 将后台返回的数据映射到form中
    * @param row{object} 节点属性
    * @param form{object} form类型
    */
    defineKeys(row,form,formName) {
      var vm = this,
        fms = form, // 应对切换源
        dir = fms.dir,
        leaf = fms.leaf;

      var key = row.id,
        pkey = row.pid || 'root';
      //忽略的节点不放入到forms的基础数据中
      if(!vm.ignoreNodes[formName].includes(row.id)) {
        dir[pkey].push(row.id);
        dir[key]==undefined && row.isLeaf!=true && vm.$set(dir, key, []);
        leaf[key]==undefined && vm.$set(leaf, key, []);
      }

      if(row.children) {
        row.children.map(function(subItem){
          // 递归处理子节点
          vm.reactForms(subItem,form,formName);
        });
      }
    },
    /**
    * 更新父级数据
    * @param id {number} 节点id
    * @param pid {number} 父节点id
    * @param form {object} form类型
    */
    updateParent(id,pid,form) {// 更新父级数据
      var vm = this,
        fms = form, // 应对切换源
        dir = fms.dir,
        leaf = fms.leaf,
        isChecked = leaf[id] && leaf[id].length==dir[id].length && leaf[id].length>0;
      
      if(isChecked) {
        !leaf[pid||'root'].includes(id) && leaf[pid||'root'].push(id);
      }else{
        var idx = leaf[pid||'root'].indexOf(id)
        idx>=0 && leaf[pid||'root'].splice(idx,1)
      }
    },
    clearUpEmpty(form) {
    	var dir = form.dir;
    	for(var key in dir) {
    		// 如果dir没有子节点数据，移除其他有该dir的关联关系
    		if(dir[key].length==0) {
        		for(var code in dir) {
            		var idx = dir[code].indexOf(key);
            		idx>=0 && dir[code].splice(idx,1);
            	}
    		}
    	}
    },
    /**
    * 全选勾选状态变化处理
    * @param val val为布尔值时，属于递归更新；为空时为点击触发
    * @param pid {number} 父节点id
    * @param form {object} form类型
    */
    handerMainChange(val,id,form){
      var vm = this,
        fms = form, // 应对切换源
        dir = fms.dir,
        leaf = fms.leaf,
        oChecked = dir[id] && leaf[id].length==dir[id].length && leaf[id].length>0,//点击前的勾选状态
        isChecked = val===''?!oChecked:val,
        key = id;
      if(leaf[key] != undefined) {
        leaf[key] = (isChecked?Object.assign([],dir[key]):[]);
        
        dir[key] && dir[key].map(function(item){// 递归子孙节点
          if(leaf[item]) {
            vm.handerMainChange(isChecked,item,form)
          }
        });
      }
    },
    /**
    * 半选状态级联更新
    * @param id {number} 节点id
    * @param form {object} form类型
    */
    indeterminate(id,form) {
      var vm = this,
        fms = form, // 应对切换源
        dir = fms.dir,
        leaf = fms.leaf,
        isMiddle = leaf[id] && 0 < leaf[id].length && leaf[id].length < dir[id].length;
      // 递归判断子节点是否有半选状态
      var isSubMiddle = false;
      dir[id] && dir[id].map(function(item){
        if(dir[item] && vm.indeterminate(item,form)) {
          isSubMiddle = true;
        }
      });

      return isMiddle || isSubMiddle;
    },
    /**
    * 判断是否为全选状态
    * @param id {number} 节点id
    * @param form {object} form类型
    */
    isAllChecked(id,form){// 判断是否为全选状态
      var vm = this,
        fms = form, // 应对切换源
        dir = fms.dir,
        leaf = fms.leaf;

      return leaf[id] && leaf[id].length == dir[id].length && leaf[id].length > 0;
    },
    ckboxClick(node,formName,isDir) {
      var vm = this, val = '', cas = vm.cascade || [];
      // 存在联动关系
      if(cas.length) {
        var id = node=='root'? 'root':node.id, 
          pid = node=='root'? 'root':node.pid,
          fms = vm.forms[formName],
          dir = fms.dir, leaf = fms.leaf,
          oChecked = (dir[id] && leaf[id].length==dir[id].length && leaf[id].length>0),//点击前的勾选状态
          key = id, pkey = pid || 'root';
        
        cas.map(function(rulestr){
          var keys = rulestr.replace(/ /g,'').split('>'),
            preKey = keys[0], sufKey = keys[1];
          
          if(keys.includes(formName)) {
            if(isDir) {// 目录节点触发 -- 忽略的节点取消联动
              preKey==formName && !vm.ignoreNodes[sufKey].includes(id) && oChecked && vm.handerMainChange(true,id,vm.forms[sufKey]);  // write
              sufKey==formName && !vm.ignoreNodes[preKey].includes(id) && !oChecked && vm.handerMainChange(false,id,vm.forms[preKey]); // readonly
            }else {// 叶子节点触发
              var isCked = fms.leaf[pkey].includes(key);
              // 如果只读不包含该项，则选中
              preKey==formName && !vm.ignoreNodes[sufKey].includes(id) && isCked && !vm.forms[sufKey].leaf[pkey].includes(key) && vm.forms[sufKey].leaf[pkey].push(key);
              // 如果可写不包含改下，则选中
              sufKey==formName && !vm.ignoreNodes[preKey].includes(id) && !isCked && vm.forms[preKey].leaf[pkey].includes(key) && vm.forms[preKey].leaf[pkey].remove(key);
            }
          }
        })
      }
    },
    //获取权限勾选结果
    getResult(){
      var vm = this,
        result = {};
      
      vm.keylist.map(function(formName) {
        var leaf = vm.forms[formName].leaf,
          checks = [];

        for(var key in leaf) {// 迭代勾选项
          leaf[key].length && leaf[key].map(function(item){
            !checks.includes(item) && checks.push(item);
          });
          leaf[key].length && !leaf[key].includes(key) && key != 'root' && !checks.includes(key) && checks.push(key);
        }

        result[formName] = checks.sort(function(a,b){return a-b});
      })

      vm.result = result;

      return result;
    },
    getCheckedNodes() {
      var vm = this,
          nodes = vm.treeData,
          list = [],
          resMap = vm.getResult();

      vm.keylist.map(function(key){
        list = list.concat(resMap[key]);
      })
      
      return vm.filterNodesByResult(nodes, list, resMap)
    },
    filterNodesByResult(nodes, list, map) {
      var vm = this;

      if(nodes.length) {
        nodes = JSON.parse(JSON.stringify(nodes));

        var res = nodes.filter(function(node){
                    var bool = list.includes(node.id);

                    if(bool) {
                      vm.checkForms.map(function(form){
                        var code = form.key,
                            prop = form.prop;

                        if(map[code].includes(node.id)) {
                          node[prop] = true;
                        }
                      });
                    }

                    return bool;
                  });

        res.map(function(node){
          if(node.children) {
            node.children = vm.filterNodesByResult(node.children, list, map);
          }
        });
        
        return res;
      }else {
        return [];
      }
    },
    // 初始化是否目录属性
    processData(data) {
    	var vm = this;
    	data.map(function(row){
        	if(row.children && row.children.length) {
        		vm.processData(row.children);
        		row.isLeaf = false;
        	}else {
        		row.isLeaf = true;
        	}
        });
    },
    /* <-------- 核心方法 末 --------> */
    init() {
      var vm = this;
      
      if(vm.keylist && vm.data.length) {
        var data = vm.data || [];
        data.map(function(row){
        	row.pid = 'root';
        });
        
        vm.processData(data);
        
        vm.keylist.map(function(formName){
          vm.reactForms(data,vm.forms[formName],formName);
          //vm.clearUpEmpty(vm.forms[formName]);
        })

        vm.$nextTick(function(){
          vm.treeData = JSON.parse(JSON.stringify(data));

          if(vm.readonly == true) {
            setTimeout(function(){
              vm.queryFeature();
            }, 50);
          }
        });
      }
    }
  }
});

Vue.component('el-bulk', {
  template:`
    <div :class="bulkCls">
      <div class="el-bulk-opts">
        <div class="batchBox-seltitle">
          {{msg.title}} <span id="device_count" style="color: blue; margin-left: 10px;">( {{count}} )</span>
          <span :class="arrowCls" @click="itemsShow = !itemsShow" style="font-size: 15px;margin-left: 10px;"></span>
        </div>
        <div style="padding-right: 20px;">
          <div class="linkbuttonGroup">
            <slot name="button"></slot>
            <a class="linkbutton linkbutton_nowanna" @click="clearAll"><span>{{msg.cancel}}</span></a>
          </div>
        </div>
      </div>
      <div :class="selectedItemCls">
        <div class="list-title">{{msg.title}}
          <span style="float: right; padding-right: 15px;" @click="itemsShow=false"><i class="el-icon el-icon-close"></i></span>
        </div>
        <div style="width: calc(100% - 20px); padding-left: 10px;">
          <div class="list-body-title">
            <span>{{msg.subTitle}}</span>
            <div title="Delete All" style="padding-left: 50px; display: flex;align-items: center;" @click="clearAll">
              <i class="el-icon el-icon-operation-delete"></i>{{msg.clear}}
            </div>
          </div>
        </div>
        <div class="list-body" style="flex: 1 1 auto;" v-if="list">
          <div v-for="item of list" class="list-item-info">
            <span class="list-item-txt">{{item[showKey]}}</span>
            <span class="list-item-op" @click="clear(item)"><i style="font-size: 14px;" class="el-icon el-icon-circle-close"></i></span>
          </div>
        </div>
      </div>
      <div class="expand-arrow" @click="expanded = !expanded">
        <i :class="expandArrowCls"></i>
      </div>
    </div>
  `,
  props: {
    list: {
      type: Array,
      default() {
        return [];
      }
    },
    target: {
      type: String,
      default: null
    },
    rowKey: {
      type: String,
      default: 'code'
    },
    showProp: {
      type: String,
      default: ''
    },
    message: {
      type: Object,
      default() {
        return {};
      }
    }
  },
  data() {
    return  {
      itemsShow: false,
      expanded: true
    }
  },
  computed: {
    msg() {
      var mgs = {
        title: vueComponetsMsg.selected,
        subTitle: vueComponetsMsg.name,
        clear: vueComponetsMsg.clear,
        cancel: vueComponetsMsg.cancel
      };

      if(this.message) {
        Object.assign(mgs, this.message);
      }

      return mgs;
    },
    showKey() {
      return this.showProp || this.rowKey;
    },
    bulkCls() {
      var vm = this;

      return {
        'el-bulk': true,
        show: vm.list.length>0,
        expand: vm.expanded
      }
    },
    selectedItemCls() {
      var vm = this;
      return {
        'selected-items-info': true,
        'show': vm.itemsShow
      }
    },
    arrowCls() {
      var vm = this;
      return {
        'el-icon': true,
        'el-icon-circle-up': vm.itemsShow,
        'el-icon-circle-down': !vm.itemsShow
      }
    },
    expandArrowCls() {
      var vm = this;
      return {
        'el-icon': true,
        'el-icon-down': vm.expanded,
        'el-icon-up': !vm.expanded
      }
    },
    count() {
      if(this.list) {
        return this.list.length;
      }else {
        return 0;
      }
    }
  },
  watch: {
    list(val){
      if(val && val.length) {

      }else {
        this.hide()
      }
    },
    expanded(val,old) {
      if(val==false) this.itemsShow = false
    }
  },
  methods: {
    hide() {
      this.itemsShow = false;
      this.expanded = true;
    },
    clear(row) {
      var vm = this,
          tar = vm.getTarget();

      if(tar) {
        var keyVal = row[vm.rowKey],
            ids = vm.list.map(function(item){ return item[vm.rowKey];}),
            tData = tar.getData(),
            trow = tData.filter(function(item){ return item[vm.rowKey] == keyVal})[0];

        vm.list.splice(ids.indexOf(keyVal),1);
        if(trow) {
          tar.toggleRowSelection(trow, false);
        }else {
          var selection = tar.$refs.ctableInner.store.states.selection,
              irow = selection.filter(function(item){ return item[vm.rowKey] == row[vm.rowKey];})[0];
          tar.toggleRowSelection(irow, false);
        }
        tar.ckList.splice(tar.ckList.indexOf(keyVal),1);
      }

      if(tar.ckList.length==0) {
        vm.hide();
      }
    },
    clearAll() {
      var vm = this,
          tar = vm.getTarget();

      vm.hide();
      tar && tar.clearSelection();
    },
    getTarget() {
      if(this.target) {
        return document.querySelector('#'+this.target).parentNode.parentNode['__vue__'];
      }else {
        return null;
      }
    }
  }
});

Vue.component('el-filter', {
  template: `
    <el-popover @show="popShow">
      <span slot="reference" :class="filterCls" @click="click"></span>
      <div>
        <div style="border-bottom: 1px solid #e9e9e9;padding-bottom: 5px;margin-bottom: 5px;">
          <el-checkbox v-model="checkAll" @change="allChange" :indeterminate="isIndeterminate">ALL</el-checkbox>
        </div>
        <el-checkbox-group v-model="checkedFields" class="column-flex" @change="fieldsChange">
          <el-checkbox v-for="item in fields" :label="item.value">{{item.text}}</el-checkbox>
        <el-checkbox-group>
      </div>
    </el-popover>
  `,
  props: {
    data: {
      type: Array,
      default: []
    },
    field: String,
    url: String,
    queryParams: {// 关联表格的查询参数
      type: Object,
      default: function() {
        return {};
      }
    },
    filterParams: {// 过滤项的请求参数
      type: Object,
      default: function() {
        return {};
      }
    },
    dataField: String,
    link: String // 与高级查询项的关联
  },
  data() {
    var vm = this,
      list = [];
    
    (vm.data || []).map(function(item){
      if(typeof item == 'object') {
        list.push(item.value);
      }else {
        list.push(item)
      }
    });

    if(vm.link) { // 高级查询中有选择项时
      list = list.filter(function(item){
        return item == vm.link;
      })
    }else if(vm.queryParams[vm.field]) {// 高级查询中无对应选择项，查询参数带过滤值时
      var defaultCk = vm.queryParams[vm.field].split(',');

      list = list.filter(function(item){
        return defaultCk.includes(item);
      })
    }
    
    return {
      checkedFields: list,
      checkAll: true
    }
  },
  computed: {
    fields() {
      var vm = this,
        list = [];
    
      (vm.data || []).map(function(item){
        if(typeof item == 'object') {
          list.push(item);
        }else {
          list.push({
            value: item,
            text: item
          })
        }
      });

      if(vm.link) {
        list = list.filter(function(item){
          return (item.value+'-') == (vm.link+'-');
        })
      }

      vm.checkAll = true;

      return list;
    },
    isIndeterminate() {
      if(this.checkedFields.length == this.fields.length) {
        return false;
      }else {
        return true
      }
    },
    filterCls() {
      var vm = this,
        isSelected = vm.fields.length > 0 && vm.fields.length != vm.checkedFields.length;
      
      return {
        'icon-filter': true,
        'filter-opt': true,
        selected: isSelected
      }
    }
  },
  watch: {
    fields(list) {
      var vm = this,
        checks = list.map(function(item){
          return item.value;
        });
    
      if(vm.queryParams[vm.field]) {// 查询参数带过滤值时
        var defaultCk = (vm.queryParams[vm.field]+'').split(',');

        checks = checks.filter(function(item){
          return defaultCk.includes(item+'');
        })

        if(checks.length == 0 && vm.fields[0]) {
          checks = [vm.fields[0].value];
        }
      }

      this.checkedFields = checks;
    },
    checkedFields(val) {
      this.$emit('change',this.field, val);
    }
  },
  methods: {
    click(evt) {
      evt.stopPropagation();
    },
    allChange(val) {
      var vm = this,
        orKeys = vm.fields.map(function(item){
          return item.value;
        });

      vm.checkedFields = val ? orKeys : [vm.fields[0].value];

      if(orKeys.length==1) vm.checkAll = true;
      vm.queryParams[vm.field] = vm.checkedFields.join(',');
    },
    fieldsChange(value) {
      var vm = this,
        checkedCount = value.length;

      vm.checkAll = checkedCount === vm.fields.length;

      if(vm.fields.length==1) {
        vm.checkedFields = [vm.fields[0].value];
        vm.checkAll = true;
      }

      if(vm.checkedFields.length == 0) {
        vm.checkedFields = [vm.fields[0].value];
      }
      vm.queryParams[vm.field] = vm.checkedFields.join(',');
    },
    popShow() {
      var vm = this;

      if(vm.url) {
        axios.post(vm.url, stringify(vm.filterParams)).then(function(res){
          if(res.data) {
            var key = vm.dataField || vm.field;
            vm.data = res.data[key];
          }
        }).catch(function(){});
      }
    }
  }
});

/**
 * 关键Array类属性说明：
 * 
 * tabs{lable、text，groups}: 属性tabs对象数组中，label是作为基础关键key，应用于表单、查询、refs的扩展标识（key_index）
 * 
 * groups{columns、url、queryParams、data、title、single、requried、rowkey,rules}: tab里的各表格分组，single是配置数据单选模式, rules是配置校验规则
 * 
 * columns{label、prop、formatter}：表格的列
 * 
 * 数据存储：
 * 
 * tabForms: 根据 key_index 动态映射各个表格选择结果记录
 * queryForm: 根据 key_index 动态映射查询条件
 * 
 * */
  Vue.component('el-tablist', {
    template: `
      <el-form ref="form" :model="tabForms" v-show="checkedKeys.length>0">
        <div v-show="!readonly" style="position: relative; padding-right: 30px;">
          <span style="position:absolute;right: 50px;z-index: 100;top: 5px;">{{msg.selectedText}} ( <font color="blue">{{seletctedCount}}</font> )</span>
          <el-popover placement="left-start">
            <span slot="reference" class="el-icon el-icon-circle-down" style="font-size: 16px;position:absolute;top: 4px;right: 30px;z-index: 100;"></span>

            <div class="selected-title">
              <span style="font-size: 14px;font-weight: bold;">{{msg.title}}</span><span onclick="document.body.click()" class="el-icon el-icon-close"></span>
            </div>
            <el-tabs v-model="activeName" class="no-color no-border tab-normal-size" style="margin-top: 5px; width: 600px;">
              <el-tab-pane v-for="item in tabList" v-if="checkedKeys.includes(item.label)" :label="item.text" :name="item.label">
                <template v-for="(group,index) in item.groups||[]">
                  <hr v-if="index>0" style="border: 1px solid #e9e9e9; border-bottom: none;margin: 10px 0 0;">
                  <!--多选模式-->
                  <el-ctable v-if="group.single!==true" height="260px" :data="tabForms[item.label+'_'+index]" 
                    :composite="true"
                    :url="Array.isArray(group.url)?group.url[1]:''"
                    :query-params="Array.isArray(group.queryParams)?group.queryParams[1]:''" 
                    :row-key="group.rowkey"
                    :keyindex="item.label+'_'+index"
                    :front-pagination="true"
                    @load-success="loadSuccess">
                    <el-table-column v-for="col in (group.selectedColumns || group.columns)" :label="col.label" :prop="col.prop"></el-table-column>
                    <el-table-column width="1" :show-overflow-tooltip="false">
                      <template slot-scope="scope">',
                        <div class="opts-wrapper" style="overflow: hidden; border-radius: 18px;">
                          <i @click="deleteChecked(scope.row,item.label+'_'+index)" class="el-icon el-icon-circle-close" style="font-size:18px;"> </i>
                        </div>
                      </template>
                    </el-table-column>
                    <!-- 查询 toolbar -->
                    <template slot="toolbar">
                      <div style="display: flex;align-items:center;">
                        <span style="color: #333;font-weight: bold;font-size: 12px;">{{group.title||msg.selected}}</span>
                        <div style="flex: 1 auto;border:1px solid #DEDFE6;border-radius:2px;margin-left: 100px;">
                          <el-input class="pairgrid-query" size="small" style="width: 90%;"></el-input>
                          <i class="el-icon el-icon-common-search"></i>
                        </div>
                        <div style="padding-left: 15px;">
                          <div @click="deleteAll(index)" class="clear-all el-icon el-icon-operation-delete"></div>
                        </div>
                      </div>
                    </template>
                  </el-ctable>
                  <!--单选模式-->
                  <el-ctable v-if="group.single===true" :data="tabForms[item.label+'_'+index]" 
                    :composite="true"
                    :url="Array.isArray(group.url)?group.url[1]:''"
                    :query-params="Array.isArray(group.queryParams)?group.queryParams[1]:''" 
                    :row-key="group.rowkey"
                    :keyindex="item.label+'_'+index"
                    :pagination="true" :rownumber="false"
                    @load-success="loadSingleSuccess">
                    <el-table-column v-for="col in (group.selectedColumns || group.columns)" :label="col.label" :prop="col.prop"></el-table-column>
                    <el-table-column width="1" :show-overflow-tooltip="false">
                      <template slot-scope="scope">',
                        <div class="opts-wrapper" style="overflow: hidden; border-radius: 18px;">
                          <i @click="deleteCheckedFile(scope.row,item.label+'_'+index)" class="el-icon el-icon-circle-close" style="font-size:18px;"> </i>
                        </div>
                      </template>
                    </el-table-column>
                    <!-- 查询 toolbar -->
                    <template slot="toolbar">
                      <div style="display: flex;align-items:center;">
                        <span style="color: #333;font-weight: bold;font-size: 12px;">{{group.title||msg.selected}}</span>
                      </div>
                    </template>
                  </el-ctable>
                </template>
              </el-tab-pane>
            </el-tabs>
          </el-popover>
        </div>

        <el-tabs v-model="activeName" v-show="tabList.length>0" class="no-color tab-normal-size" type="card" :closable="!readonly" style="padding-left: 35px;padding-right: 30px;"
          @tab-remove="removeTab">
          <el-tab-pane style="padding-left: 10px;padding-top: 10px;padding-right: 30px;" 
            v-for="item in tabList" v-if="checkedKeys.includes(item.label)" :label="item.text" :name="item.label">
            <div v-for="(group,index) in item.groups||[]" class="group" :label="group.title||msg.selected">
              <!--多选模式-->
              <el-form-item v-if="group.single!==true" label="" style="border: 1px solid #e9e9e9;" :prop="item.label+'_'+index" :rules="readonly==true?'':group.rules">
                <el-ctable :ref="item.label+'_'+index" height="300px" :keyindex="item.label+'_'+index" :key="item.label+'_'+index"
                  :data="group.data" 
                  :url="Array.isArray(group.url)?group.url[0]:group.url" 
                  :query-params="Array.isArray(group.queryParams)?group.queryParams[0]:group.queryParams" :row-key="group.rowkey"
                  @selection-change="selectChange">
                  <el-table-column v-if="!readonly" type="selection" width="45" :reserve-selection="true"></el-table-column>
                  <el-table-column v-for="col in group.columns" :label="col.label" :prop="col.prop">
                    <template slot-scope="scope">
                      <div v-html="columnFormatter(scope.row,col,scope.row[col.prop])"></div>
                    </template>
                  </el-table-column>
                  <!-- 查询 toolbar -->
                  <template slot="toolbar">
                    <div class="el-query" style="width: fit-content; position: relative;">
                      <div class="advanceQuery" style="width: fit-content; position: relative;">
                        <div class="pairgrid-query el-input el-input--small">
                          <input v-model="queryForm[item.label+'_'+index]" class="el-input__inner">
                        </div>
                        <i class="el-icon el-icon-common-search" @click="query(item.label+'_'+index)" style="margin-left: 10px;"></i>
                      <div>
                    </div>
                  </template>
                </el-ctable>
              </el-form-item>
              <!--单选模式-->
              <el-form-item v-if="group.single===true" label="" style="border: 1px solid #e9e9e9;" :prop="item.label+'_'+index" :rules="readonly==true?'':group.rules">
                <el-ctable :ref="item.label+'_'+index" height="260px" :keyindex="item.label+'_'+index" :key="item.label+'_'+index"
                  :data="group.data" 
                  :url="Array.isArray(group.url)?group.url[0]:group.url" 
                  :query-params="Array.isArray(group.queryParams)?group.queryParams[0]:group.queryParams" :row-key="group.rowkey"
                  @current-change="currentChange">
                  <el-table-column v-if="!readonly" label="Select" width="60" :show-overflow-tooltip="false">
                    <template slot-scope="scope">
                      <div class="tableDiv el-icon el-icon-status-yes selected-status" ></div>
                    </template>
                  </el-table-column>
                  <el-table-column v-for="col in group.columns" :label="col.label" :prop="col.prop"></el-table-column>
                  <!-- 查询 toolbar -->
                  <template v-if="!readonly" slot="toolbar">
                    <div class="el-query" style="width: fit-content; position: relative;">
                      <div class="advanceQuery" style="width: fit-content; position: relative;">
                        <div class="pairgrid-query el-input el-input--small">
                          <input v-model="queryForm[item.label+'_'+index]" class="el-input__inner">
                        </div>
                        <i class="el-icon el-icon-common-search" @click="query(item.label+'_'+index)" style="margin-left: 10px;"></i>
                      <div>
                    </div>
                  </template>
                </el-ctable>
              </el-form-item>
            </div>
          </el-tab-pane>
        </el-tabs>
      </el-form>
      `,
    props: {
        tabs: {
          type: Array,
          default: []
        },
        keys: {
          type: Array,
          default: []
        },
        messages: {
          type: Object,
          default: function() {
            return {}
          }
        },
        readonly: {
          type: Boolean,
          default: false
        }
    },
    data() {
        
        return {
          activeName: this.tabs[0]? this.tabs[0].label:'',
          tabForms: {

          },
          queryForm: {

          }
        }
    },
    computed: {
      tabList() {

        return this.tabs || [];
      },
      checkedKeys() {
        return this.keys || [];
      },
      seletctedCount() {
        var vm = this,
          count = 0;

        vm.tabList.map(function(item){
          item.groups.map(function(group,index){
            var key = item.label+'_'+index;
            if(vm.tabForms[key] && vm.checkedKeys.includes(item.label)) {
              count += vm.tabForms[key].length;
            }
          })
        });

        return count;
      },
      msg() {
        return Object.assign({
          title: 'Selected',
          selectedText: 'Selected',
          placeholder: '',
          confirm: 'Confirm',
          clear: 'Sure to clear all?'
        },this.messages);
      }
    },
    watch: {
        tabList: function(list) {
          var vm = this,
            keys = vm.checkedKeys;
          
        vm.initTabForms();
          if(vm.activeName) {
            if(!keys.includes(vm.activeName)) vm.activeName = keys[0];
          }else {
            vm.activeName = keys[0];
          }
        },
        checkedKeys(val,oldVal) {
          var vm = this,
              keys = val||[],
              oldKeys = oldVal||[];
        
          if(vm.activeName) {
            if(!keys.includes(vm.activeName)) vm.activeName = keys[0];
          }else {
            vm.activeName = keys[0];
          }
          /**
          * 对比新旧数据判定那个tab中表格被重绘，将历史数据压入store中（回显同逻辑）
          **/
          vm.tabList.map(function(tab){
            var key = tab.label;
            if(!keys.includes(key)) {
              (tab.groups||[]).map(function(group,index) {
                //vm.tabForms[key+'_'+index] = [];

              })
            }else if(!oldKeys.includes(key)) {
              vm.$nextTick(function(){
                (tab.groups||[]).map(function(group,index) {
                  var propKey = tab.label+'_'+index;
                  vm.tabForms[propKey].map(function(row) {
                    vm.$refs[propKey][0].toggleRowSelection(row, true);
                  });
                })
              })
            }
          })
        }     
    },
    methods: {
      initTabForms() {
        var vm = this;

        vm.tabList.map(function(item){
          (item.groups||[]).map(function(group,index){
            var key = item.label+'_'+index;
            if(vm.tabForms[key] === undefined) {
              Vue.set(vm.tabForms,key, []);
              Vue.set(vm.queryForm,key, '');
            }
          })
        });
      },
      fetchData(group, index) {
        console.log(group, index)
      },
      query(code) {
        var vm = this,
          tb = vm.$refs[code][0];
        // 表格查询
        if(tb.queryParams) tb.queryParams.search_text = vm.queryForm[code];
      },
      columnFormatter(row,col,val) {
        var vm = this,
          fmt = col.formatter;

        if(typeof fmt == 'function') {
          return fmt(row,col,val);
        }

        return val;
      },
      removeTab(code) {
        var vm = this;
        
        if(vm.keys) {
          vm.keys.splice(vm.keys.indexOf(code),1);
        }
      },
      loadSingleSuccess(res, ev, t) {
        var vm = this,
            key = t.$el.getAttribute('keyindex'),
            oTb = vm.$refs[key][0],
            rowKey = oTb.rowKey;
        // 回显勾选逻辑: 单选模式
        if(rowKey && res && res.rows) {
          var tRows = oTb.getData();

          oTb.$refs.ctableInner.store.states.currentRow = '';
          // 回显选中，似乎store中预先有数据时，渲染时才会自动选中匹配项
          res.rows.map(function(row) {
            var srow = row;
            if(tRows.length) {
              var frow = tRows.filter(function(item){
                  return item[rowKey] == row[rowKey]
                })[0];
              
              frow && (srow = frow);
            }

            oTb.setCurrentRow(srow);
          });
        }
      },
      loadSuccess(res, ev, t) {
        var vm = this,
            key = t.$el.getAttribute('keyindex'),
            oTb = vm.$refs[key][0],
            rowKey = oTb.rowKey;
        // 回显勾选逻辑: 需要兼容多选、单选模式
        if(rowKey && res && res.rows) {
          var selection = oTb.$refs.ctableInner.store.states.selection,
            skeys = selection.map(function(item){ return item[rowKey]}),
            tRows = oTb.getData();
          
          oTb.clearSelection();
          // 回显选中，似乎store中预先有数据时，渲染时才会自动选中匹配项
          res.rows.map(function(row) {
            var srow = row;
            if(tRows.length) {
              var frow = tRows.filter(function(item){
                  return item[rowKey] == row[rowKey]
                })[0];
              
              frow && (srow = frow);
            }

            oTb.toggleRowSelection(srow, true);
          });
        }
      },
      selectChange(s,t) {
        var vm = this,
          key = t.$el.getAttribute('keyindex');
        
        vm.tabForms[key] = s;

        vm.$emit('select-change',vm.getChecked());
      },
      currentChange(s,o,t) {
        var vm = this,
          key = t.$el.getAttribute('keyindex');
        
        if(s) {
          vm.tabForms[key] = [s];
        }else {
          vm.tabForms[key] = [];
        }
        
        vm.$emit('select-change',vm.getChecked());
      },
      deleteChecked(row,keyindex) {
        var vm = this
          key = keyindex,
          tb = vm.$refs[key][0],
          form = vm.tabForms[key];
        var selection = tb.$refs.ctableInner.store.states.selection;

        selection.map(function(item){
          if(item[tb.rowKey] == row[tb.rowKey]) {
            tb.toggleRowSelection(item,false);
          }
        })
      },
      deleteCheckedFile(row,keyindex) {
        var vm = this,
          key = keyindex,
          tb = vm.$refs[key][0],
          form = vm.tabForms[key];
        
        tb.setCurrentRow(null);
      },
      deleteAll(index) {
        var vm = this,
            key = vm.activeName+'_'+index,
            tb = vm.$refs[key][0];

        if(vm.tabForms[key] && vm.tabForms[key].length) {
          vm.$confirm(vm.msg.clear,vm.msg.confirm).then(function(r){
            if(r) tb.clearSelection();
          })
        }
      },
      getChecked() {
        var vm = this,
            form = {

            };

        for(var key in vm.tabForms) {
					var code = key.substring(0,key.lastIndexOf('_'));

					if(vm.checkedKeys.includes(code)) form[key] = vm.tabForms[key];
				}

        return form;
      },
      validate() {
        var vm = this,
            isValid = true;
            
        this.$refs.form.validate(function(valid){
          isValid = valid
        });

        return isValid;
      }
    },
    mounted() {
      this.initTabForms();
    }
  })