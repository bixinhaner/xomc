//自定义密码输入框//input元素光标操作
class CursorPosition {
  constructor(_inputEl) {
    this._inputEl = _inputEl;
  }
  //获取光标的位置 前，后，以及中间字符
  get() {
    var rangeData = { text: "", start: 0, end: 0 };
    if (this._inputEl.setSelectionRange) {
      // W3C
      this._inputEl.focus();
      rangeData.start = this._inputEl.selectionStart;
      rangeData.end = this._inputEl.selectionEnd;
      rangeData.text =
        rangeData.start != rangeData.end
          ? this._inputEl.value.substring(rangeData.start, rangeData.end)
          : "";
    } else if (document.selection) {
      // IE
      this._inputEl.focus();
      var i,
        oS = document.selection.createRange(),
        oR = document.body.createTextRange();
      oR.moveToElementText(this._inputEl);
      rangeData.text = oS.text;
      rangeData.bookmark = oS.getBookmark();
      for (
        i = 0;
        oR.compareEndPoints("StartToStart", oS) < 0 &&
        oS.moveStart("character", -1) !== 0;
        i++
      ) {
        if (this._inputEl.value.charAt(i) == "\r") {
          i++;
        }
      }
      rangeData.start = i;
      rangeData.end = rangeData.text.length + rangeData.start;
    }
    return rangeData;
  }
  //写入光标的位置
  set(rangeData) {
    var oR;
    if (!rangeData) {
      console.warn("You must get cursor position first.");
    }
    this._inputEl.focus();
    if (this._inputEl.setSelectionRange) {
      //  W3C
      this._inputEl.setSelectionRange(rangeData.start, rangeData.end);
    } else if (this._inputEl.createTextRange) {
      // IE
      oR = this._inputEl.createTextRange();
      if (this._inputEl.value.length === rangeData.start) {
        oR.collapse(false);
        oR.select();
      } else {
        oR.moveToBookmark(rangeData.bookmark);
        oR.select();
      }
    }
  }
}

Vue.component('el-password', {
  template: `
  <div
    class="el-password el-input"
    :class="[size ? 'el-input--' + size : '', { 'is-disabled': disabled }]"
  >
    <input
      class="el-input__inner"
      :placeholder="placeholder"
      ref="input"
      :style="{ paddingRight: padding + 'px' }"
      :disabled="disabled"
      :readonly="readonly"
      :maxlength="maxlength"
      @keyup.enter="handEnter"
      @focus="handleFocus"
      @blur="handleBlur"
      @input="handleInput"
      @change="change"
      @compositionstart="handleCompositionStart"
      @compositionend="handleCompositionEnd"
    />
    <div class="password-tools">
      <i
        v-if="clearable !== false"
        v-show="pwd !== '' && isfocus"
        @mousedown.prevent
        class="el-input__icon el-icon-circle-close el-input__clear"
        @click="clearValue"
      ></i>
      <i
        v-if="showPassword !== false"
        v-show="pwd !== '' || isfocus"
        class="el-input__icon el-icon-view el-input__clear"
        @click="changePasswordShow"
      ></i>
    </div>
  </div>
  `,
  props: {
    value: { default: "" },
    size: { type: String, default: "" },
    maxlength: { default: "" },
    placeholder: { type: String, default: "请输入" },
    disabled: { type: [Boolean, String], default: false },
    readonly: { type: [Boolean, String], default: false },
    clearable: { type: [Boolean, String], default: false },
    showPassword: { type: [Boolean, String], default: false },
  },
  data() {
    return {
      symbol: "●", //自定义的密码符号
      pwd: "", //密码明文数据
      padding: 15,
      show: false,
      isfocus: false,
      inputEl: null, //input元素
      isComposing: false, //输入框是否还在输入（记录输入框输入的是虚拟文本还是已确定文本）
    };
  },
  mounted() {
    this.inputEl = this.$refs.input;
    this.pwd = this.value;
    this.inputDataConversion(this.pwd);
  },
  watch: {
    value: {
      handler: function (value) {
        if (this.inputEl) {
          this.pwd = value;
          this.inputDataConversion(this.pwd);
        }
      },
    },
    showPassword: {
      handler: function (value) {
        let padding = 15;
        if (value) {
          padding += 18;
        }
        if (this.clearable) {
          padding += 18;
        }
        this.padding = padding;
      },
      immediate: true,
    },
    clearable: {
      handler: function (value) {
        let padding = 15;
        if (value) {
          padding += 18;
        }
        if (this.showPassword) {
          padding += 18;
        }
        this.padding = padding;
      },
      immediate: true,
    },
  },
  methods: {
    select() {
      this.$refs.input.select();
    },
    focus() {
      this.$refs.input.focus();
    },
    blur() {
      this.$refs.input.blur();
    },
    handEnter(event) {
      this.$emit("enter", event);
      event.preventDefault();
      event.stopPropagation();
      return false;
    },
    handleFocus(event) {
      this.isfocus = true;
      this.$emit("focus", event);
    },
    handleBlur(event) {
      this.isfocus = false;
      this.$emit("blur", event);
      //校验表单
      this.$emit("ElFormItem", "el.form.blur", [this.value]);
    },
    change(...args) {
      this.$emit("change", ...args);
    },
    clearValue() {
      this.pwd = "";
      this.inputEl.value = "";
      this.$emit("input", "");
      this.$emit("change", "");
      this.$emit("clear");
      this.$refs.input.focus();
    },
    changePasswordShow() {
      this.show = !this.show;
      this.inputDataConversion(this.pwd);
      this.$refs.input.focus();
    },
    inputDataConversion(value) {
      //输入框里的数据转换，将123转为●●●
      if (!value) {
        this.inputEl.value = "";
        return;
      }
      let data = "";
      for (let i = 0; i < value.length; i++) {
        data += this.symbol;
      }
      //使用元素的dataset属性来存储和访问自定义数据-*属性 (存储转换前数据)
      this.inputEl.dataset.value=  this.pwd;
      this.inputEl.value = this.show ? this.pwd : data;
    },
    pwdSetData(positionIndex, value) {
      //写入原始数据
      let _pwd = value.split(this.symbol).join("");
      if (_pwd) {
        let index = this.pwd.length - (value.length - positionIndex.end);
        this.pwd =
          this.pwd.slice(0, positionIndex.end - _pwd.length) +
          _pwd +
          this.pwd.slice(index);
      } else {
        this.pwd =
          this.pwd.slice(0, positionIndex.end) +
          this.pwd.slice(positionIndex.end + this.pwd.length - value.length);
      }
    },
    handleInput(e) {
      //输入值变化后执行 //撰写期间不应发出输入
      if (this.isComposing) return;
      let cursorPosition = new CursorPosition(this.inputEl);
      let positionIndex = cursorPosition.get();
      let value = e.target.value;
      //整个输入框的值
      if (this.show) {
        this.pwd = value;
      } else {
        this.pwdSetData(positionIndex, value);
        this.inputDataConversion(value);
      }
      cursorPosition.set(positionIndex, this.inputEl);
      this.$emit("input", this.pwd);
    },
    handleCompositionStart() {
      //表示正在写
      this.isComposing = true;
    },
    handleCompositionEnd(e) {
      if (this.isComposing) {
        this.isComposing = false;
        //handleCompositionEnd比handleInput后执行，避免isComposing还为true时handleInput无法执行正确逻辑
        this.handleInput(e);
      }
    },
  },
})

Vue.component('el-popfilter', {
  template:`
    <div v-if="visible" class="pop-filter-wrap">
      <el-popover @show="checkPopoverShow" @hide="checkPopoverHide">
        <div class="pop-filter-box">
            <div style="flex: 1 auto;border:1px solid #DEDFE6;border-radius:2px;margin-bottom: 10px;">
                <el-input class="pairgrid-query"
                    v-model="searchText"
                    @keyup.enter.native="query"
                    :placeholder="label"
                    size="mini"
                    style="padding-left: 5px;width: calc(99% - 5px);max-width: none;">
                    <i slot="suffix" @click="query" class="el-icon el-icon-common-search" style="margin-top: 6px;"></i>
                </el-input>
            </div>
            <!-- multiple opts -->
            <div v-if="isMultiple" style="display: flex; align-items: baseline">
                <el-checkbox :indeterminate="isIndeterminate" v-model="isCheckAll" @change="checkAll" style="margin-bottom: 0px;"> ( {{messages.selectAll}}</el-checkbox>
                <div style="margin: 0 5px;font-size: 12px;">| <span @click="checkReverse" style="cursor: pointer;">{{messages.reverse}}</span> )</div>
                <div class="el-table">
                    <span class="caret-wrapper" :class="{'ascending': sort == 'up', 'descending': sort == 'down' }" style="height: 20px;">
                        <i class="sort-caret ascending" @click="sortChange('up')"></i>
                        <i class="sort-caret descending" @click="sortChange('down')" style="bottom: 0px;"></i>
                    </span>
                </div>
            </div>
            <!-- single opts -->
            <div v-if="!isMultiple" style="display: flex; align-items: center;">
                <div style="padding: 0px 5px;">{{messages.all}}</div>
                <div class="el-table">
                    <span class="caret-wrapper" :class="{'ascending': sort == 'up', 'descending': sort == 'down' }" style="height: 20px;">
                        <i class="sort-caret ascending" @click="sortChange('up')"></i>
                        <i class="sort-caret descending" @click="sortChange('down')" style="bottom: 0px;"></i>
                    </span>
                </div>
            </div>
            <div style="margin:8px 0px;height:1px;background:#E9EDF9;"></div>
            <!-- list -->
            <el-checkbox-group v-if="isMultiple" v-model="itemChecked" @change="handleCheckedChange">
                <el-checkbox v-for="item in checkList" :key="item.value" :label="item.value">{{item.label}}</el-checkbox>
            </el-checkbox-group>
            <div v-if="!isMultiple" style="padding-bottom: 5px;max-height: 300px;overflow: auto;">
                <div
                    v-for="item in checkList"
                    :key="item.value"
                    :class="{'pop-filter-item': true, 'selected': item.value === itemChecked}"
                    @click="singleSelect(item.value)"
                >{{item.label}}</div>
            </div>
            <div class="buttonGroup">
                <el-button size="mini" type="primary" @click="checkOk">{{messages.ok}}</el-button>
                <el-button size="mini" @click="checkCancel">{{messages.cancel}}</el-button>
            </div>
        </div>
        <div slot="reference" :class="popoverShow == true ?'pop-filter-icon-box filter-icon-box-active' : 'pop-filter-icon-box'" style="white-space: nowrap;">
            <span style="display: inline-block;min-width: 20px;">{{label}}</span>
            <span v-if="checkedLabels.length" style="zoom: 0.8;">: {{checkedLabels}}</span>
            <i class="el-icon el-icon-down" style="margin-left:5px;"></i>
            <span v-if="closable === true" @click="hidePopfilter" class="el-icon el-icon-close" style="margin-left:10px;"></span>
        </div>
      </el-popover>
    </div>
  `,
  model: {
    prop: 'checked',
    event: 'check-change'
  },
  props: {
    list: {
      type: Array,
      default() {
        return [];
      }
    },
    label: {
        type: String,
        default: ''
    },
    checked: {
      type: Array,
      default() {
          return []
      }
    },
    visible: {
        type: Boolean,
        default: true
    },
    closable: {
        type: Boolean,
        default: false
    },
    type: {
        type: String,
        default: 'multiple'
    }
  },
  data() {

    return  {
        searchText: '',
        sort: '',
        popoverShow: false,
        isCheckAll: false,
        oldChecked: this.checked? [...this.checked] : [],
        itemChecked: this.checked? [...this.checked] : [],
        messages: {
            ok: 'OK',
            cancel: 'Cancel',
            all: 'All',
            selectAll: 'All',
            reverse: 'Reverse',
        }
    }
  },
  computed: {
    /**
    * check list： 根据排序方式调整顺序，根据queryStr动态过滤展示项（）
    **/
    checkList() {
        var vm = this,
            sort = vm.sort,
            queryStr = vm.searchText.trim(),
            list = (vm.list || []).map(function(item){
                return Object.assign({}, item);
            }),
            // 查询过滤后
            filterList = list.filter(function(item){
                return !queryStr || item.label.indexOf(queryStr) > -1;
            }),
            labels = filterList.map(function(item){
                return item.label;
            }).sort();

        if(sort == 'up') {
            filterList.sort(function(pre, cur){
                return labels.indexOf(pre.label) - labels.indexOf(cur.label)
            });
        }else if(sort == 'down') {
            filterList.sort(function(pre, cur){
                return labels.indexOf(pre.label) - labels.indexOf(cur.label)
            }).reverse();
        }

        return filterList;
    },
    isIndeterminate() {
        var vm = this,
            filterChecked = [];

        vm.checkList.map(function(item){
            if(vm.itemChecked.includes(item.value)) filterChecked.push(item.value);
        })

        return filterChecked.length > 0 && filterChecked.length < vm.checkList.length;
    },
    isMultiple() {
        var vm = this;

        return vm.type !== 'single';
    },
    checkedLabels() {
      var vm = this,
          ckList = vm.itemChecked,
          filterList = (vm.list || []).filter(function(item){
            if(vm.type == 'single') {
              return item.value !== '' && ckList == item.value;
            }else {
              return item.value !== '' && ckList.includes(item.value);
            }
          }),
          ckLabels = filterList.map((item)=>{
            return item.label;
          }),
          labelStr = ckLabels.join(' 、');

      if(vm.type !== 'single' && ckLabels.length) {
        if(ckLabels[0].length <= 8 || labelStr.length <= 8) {
          labelStr = ckLabels[0];
        }else {
          labelStr = labelStr.substr(0,8) + '...';
        }

        if(ckLabels.length > 1) {
          labelStr += '(' + ckLabels.length + ')';
        }
      }else {
        if(labelStr.length > 8) labelStr = labelStr.substr(0,8) + '...';
      }

      return labelStr;
    }
  },
  watch: {
    checkList(list) {
       var vm = this,
           filterChecked = [];

       list.map(function(item){
          if(vm.type == 'single') {
            if(vm.itemChecked == item.value) filterChecked.push(item.value);
          }else {
            if(vm.itemChecked.includes(item.value)) filterChecked.push(item.value);
          }
       });

       if(list.length && filterChecked.length == list.length) vm.isCheckAll = true;
    },
    checked(list) {
      var vm = this;

      if(vm.type === 'single') {
        vm.oldChecked = list;
        vm.itemChecked = list;
      }else {
          vm.oldChecked = [...list];
          vm.itemChecked = [...list];
      }
    }
  },
  methods: {
    query() {

    },
    checkAll() {
        var vm = this,
            allItems = vm.checkList.map(function(item){
                return item.value;
            }),
            isCheckAll = vm.isCheckAll,
            oList = [...vm.itemChecked];

        // vm.itemChecked = isCheckAll ? allItems : [];
        // 当前所有项的添加和移除
        allItems.map(function(code){
            if(isCheckAll) {
                if(!oList.includes(code)) oList.push(code);
            }else {
                var idx = oList.indexOf(code);
                if(idx > -1) oList.splice(idx,1);
            }
        });

        vm.itemChecked = oList;
        vm.$emit('change', vm.itemChecked);
    },
    checkReverse() {
        var vm = this,
            allItems = vm.checkList.map(function(item){ return item.value; }),
            filterChecked = [];

        vm.checkList.map(function(item){
            if(vm.itemChecked.includes(item.value)) filterChecked.push(item.value);
        })

        var unchecked = allItems.filter(function(item){ return !filterChecked.includes(item); }),
            oList = [...vm.itemChecked];
        // 移除过滤后的勾选项
        filterChecked.map(function(code){
            var idx = oList.indexOf(code);
            if(idx > -1) oList.splice(idx,1);
        });
        // 添加过滤后的未勾选项
        unchecked.map(function(code){
            if(!oList.includes(code)) oList.push(code);
        });
        vm.itemChecked = oList;
        // vm.itemChecked = unchecked;
        vm.isCheckAll = unchecked.length == allItems.length;
        vm.$emit('change', vm.itemChecked);
    },
    handleCheckedChange() {
        var vm = this,
            filterChecked = [];

        vm.checkList.map(function(item){
            if(vm.itemChecked.includes(item.value)) filterChecked.push(item.value);
        });

        if(filterChecked.length == vm.checkList.length) {
            vm.isCheckAll = true;
        }else {
            vm.isCheckAll = false;
        }

        vm.$emit('change', vm.itemChecked);
    },
    checkCancel() {
        var vm = this;

        if(vm.type === 'single') {
          vm.itemChecked = vm.oldChecked;
        }else {
          vm.itemChecked = [...vm.oldChecked];
        }

        vm.checkPopoverHide();
        vm.$emit('change', vm.itemChecked);
    },
    checkOk() {
        var vm = this;

        if(vm.type === 'single') {
            vm.oldChecked = vm.itemChecked;
        }else {
            vm.oldChecked = [...vm.itemChecked];
        }

        vm.$emit('check-change', vm.itemChecked);
        vm.checkPopoverHide();
    },
    checkPopoverShow() {
        var vm = this;

        vm.popoverShow = true;
    },
    checkPopoverHide() {
        var vm = this;
        
        vm.popoverShow = false;
        document.body.click();
        
        vm.$nextTick(()=>{
          if(vm.type === 'single') {
            vm.itemChecked = vm.oldChecked;
          }else {
            vm.itemChecked = [...vm.oldChecked];
          }
        })
    },
    sortChange(sortBy) {
        var vm = this;

        if(vm.sort == sortBy) vm.sort = '';
        else vm.sort = sortBy;
    },
    hidePopfilter() {
        var vm = this;

        vm.visible = false;
        vm.$emit('update:visible', false);
        vm.$emit('close');
    },
    singleSelect(code) {
        var vm = this;

        vm.itemChecked = code;
    },
    reset() {
      var vm = this;

      if(vm.type === 'single') {
          vm.itemChecked = '';
          vm.oldChecked = '';
      }else {
          vm.itemChecked = [];
          vm.oldChecked = [];
      }
    }
  }
});

Vue.component('el-navigator', {
  template:`
    <div class="navigator-ctn">
      <el-tabs v-model="editableTabsValue" type="card" closable @tab-remove="removeTab" @tab-click="clickTab">
        <el-tab-pane 
          v-for="(item, index) in editableTabs"
          :key="item.url+item.id"
          :label="item.title"
          :name="item.id"
          :url="item.url"
        >
          <div class="tab-nav-content" :id="'tab_content_' + item.id">{{item.content}}</div>
        </el-tab-pane>
      </el-tabs>
    </div>
  `,
  props: {
    list: {
      type: Array,
      default() {
        return [];
      }
    }
  },
  data() {

    return  {
		editableTabsValue: '',
        editableTabs: this.list||[],
    }
  },
  computed: {
    
  },
  watch: {
    list: {
      handler: function(val){
        this.editableTabs = val||[];
      },
      deep: true
    },
  },
  methods: {
    addTab(item) {
      let vm = this,
          existIds = this.editableTabs.map(function(item){
            return item.id+'';
          }),
          existUrls = this.editableTabs.map(function(item){
            return item.url;
          });

      vm.clearSiblings(item);

      if(existIds.includes(item.id+'')) {
        let fltItem = this.editableTabs.filter(function(m){
              return m.id == item.id;
            });

        if(fltItem.length > 0) {
          fltItem[0].title = item.title;
          fltItem[0].netType = item.netType;
        }
        
        this.editableTabsValue = item.id;

        if(enableWebPerformanceMode == true) {
          this.loadContent(item.id+'', item.url, item.callback);
        }else {
          this.$nextTick(() => {
            setTimeout(function(){
              $(window).resize();
      
              try {
                let list = document.querySelectorAll('[_echarts_instance_]');
      
                if(list) {
                  Array.from(list).map(function(item){
                    echarts.getInstanceByDom(item).resize();
                  });
                }
              } catch (error) { }
            },300);
          })
        }

        return;
      }
      if(existUrls.includes(item.url)) {
        let filterItem = this.editableTabs.filter(function(m){
              return m.url == item.url;
            });

        if(filterItem.length > 0) {
          filterItem[0].title = item.title;
          filterItem[0].netType = item.netType;          
        }

        this.editableTabsValue = filterItem[0].id;

        if(enableWebPerformanceMode == true) {
          this.loadContent(filterItem[0].id+'', item.url, item.callback);
        }else {
          this.$nextTick(() => {
            setTimeout(function(){
              $(window).resize();
      
              try {
                let list = document.querySelectorAll('[_echarts_instance_]');
      
                if(list) {
                  Array.from(list).map(function(item){
                    echarts.getInstanceByDom(item).resize();
                  });
                }
              } catch (error) { }
            },300);
          })
        }

        return;
      }

      this.editableTabs.push(item);
      this.editableTabsValue = item.id;
      this.$nextTick(()=>{
        let ctner = $('#tab_content_' + item.id);

        ctner.addClass('loading');
        ctner.load(item.url,()=>{
			      $.parser.parse(ctner);

            if(item.callback && typeof item.callback == 'function') {
              try{
                item.callback();
              }catch(e){}
            }

            ctner.removeClass('loading');

            vm.clearSiblings(item);
        });
      });
    },
    // 删除tab
    removeTab(targetName) {
      var vm = this;
      let tabs = this.editableTabs;
      let activeName = this.editableTabsValue;
      
      // 查找被删除的页签信息和索引
      let removedTab = tabs.find(tab => tab.id == targetName);
      let removedIndex = tabs.findIndex(tab => tab.id == targetName);

      if(activeName == targetName) {
        tabs.forEach((tab, index) => {
          if(tab.id == targetName) {
            let nextTab = tabs[index+1] || tabs[index-1];
            if(nextTab) {
              activeName = nextTab.id;
              if(enableWebPerformanceMode == true) vm.loadContent(activeName, nextTab.url, nextTab.callback);
            }
          }
        });
      }

      this.editableTabsValue = activeName;
      
      // 使用 splice 直接修改数组，确保 Vue 能检测到变化
      if(removedIndex > -1) {
        this.editableTabs.splice(removedIndex, 1);
      }
            
      // 修复：触发自定义事件，通知父组件页签已被删除
      if(removedTab) {
        this.$emit('tab-removed', removedTab);
      }
      
      document.body.click();

      setTimeout(function(){
        $(window).resize();

        try {
          let list = document.querySelectorAll('[_echarts_instance_]');

          if(list) {
            Array.from(list).map(function(item){
              echarts.getInstanceByDom(item).resize();
            });
          }
        } catch (error) { }
      },200);
    },
    clickTab(item) {
      this.clearSiblings(item);

      if(enableWebPerformanceMode == true) {
        this.loadContent(item.name, item.$attrs.url, item.callback);
      }else {
        setTimeout(function(){
          $(window).resize();

          try {
            let list = document.querySelectorAll('[_echarts_instance_]');

            if(list) {
                Array.from(list).map(function(item){
                  echarts.getInstanceByDom(item).resize();
                });
            }
          } catch (error) { }
        },200);
      }
    },
    clearSiblings(item) {
      if(enableWebPerformanceMode == true) {
        var siblings = $('.tab-nav-content').filter(function() {
          var code = item.id || item.name;
          return $(this).attr('id') != 'tab_content_' + code;
        });  
        siblings.html('');
      }
    },
    loadContent(id, url, callback) {
      this.$nextTick(() => {
        let ctner = $('#tab_content_' + id);

        ctner.addClass('loading');
        ctner.load(url, ()=>{
			      $.parser.parse(ctner);

            if(callback && typeof callback == 'function') {
              try{
                callback();
              }catch(e){}
            }

            ctner.removeClass('loading');

            setTimeout(function(){
              $(window).resize();

              try {
                let list = document.querySelectorAll('[_echarts_instance_]');

                if(list) {
                    Array.from(list).map(function(item){
                      echarts.getInstanceByDom(item).resize();
                    });
                }
              } catch (error) { }
            },200);
        });
      });
    }
  }
});

Vue.component('el-navmenu', {
  template:`
    <div>
      <el-menu mode="horizontal" :default-active="menuActive" class="submenu-horizontal" @select="menuSelect">
        <template v-for="item in menus">
          <el-submenu v-if="hasChild(item.children)" :index="item.id" :key="item.id">
            <template slot="title">
              <i class="navmenu-icon" :class="item.menu_icon?item.menu_icon.split(' '):[]"></i>
              {{item.menu_name}}
            </template>
            <el-menu-item v-for="sub in item.children" :index="sub.menu_url" :key="sub.menu_url">
              <i class="navmenu-icon" :class="sub.menu_icon?sub.menu_icon.split(' '):[]"></i>
              {{sub.menu_name}}
            </el-menu-item>
          </el-submenu>

          <el-menu-item v-else :index="item.menu_url" :key="item.menu_url">
            <i class="navmenu-icon" :class="item.menu_icon?item.menu_icon.split(' '):[]"></i>
            {{item.menu_name}}
          </el-menu-item>
        </template>
      </el-menu>
    </div>
  `,
  props: {
	  list: {
	    type: Array,
	    default() {
	    	return [];
	    }
    },
    defaultActive: {
    	type: String,
	    default: ''
    }
  },
  data() {

    return  {
    	menuActive: this.defaultActive		
    }
  },
  computed: {
    menus() {
      let menus = this.list?this.list:[];

      return menus;
    }
  },
  watch: {
    defaultActive(val) {
      this.menuActive = val;
    }
  },
  methods: {
    hasChild(list) {
      let bool = false;

      if(list && list.length) {
        bool = true;
      }

      return bool;
    },
    menuSelect(url) {
      this.$emit('select', url);
    }
  }
});