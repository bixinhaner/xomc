function _rightRowClick(obj, idVal, idField) {
  var ctn = $(obj).parents('.couple-wrap'),
    leftTb = $('.couple-left-table', ctn),
    rightTb = $('.couple-right-table', ctn);
  // 可能还要清除缓存的选中数据
  // var tips = $.fn.pairgrid.defaults.messages.tips,
  //   msg = $.fn.pairgrid.defaults.messages.delete;
  //   $.messager.confirm(tips, msg, function(r) {
  //   if (r) {
      
  //   }
  // }).addClass("seriousConfirm");
  rightTb.datagrid('deleteRow', rightTb.datagrid('getRowIndex', idVal));

      var ckList = rightTb.data('checklist'),
        opts = leftTb.datagrid('options');
      if (ckList.includes(idVal)) {
        ckList.splice(ckList.indexOf(idVal), 1);
        // 全局数据移除该项
        var originalRows = rightTb.datagrid('getData').originalRows,
            rowKeys = originalRows.map(function(item){ return item[idField]; });
        originalRows.splice(rowKeys.indexOf(idVal), 1);
      }

      var rCks = leftTb.datagrid('getChecked');
      if (rCks) {
        var index = -1;
        rCks.map(function(row, idx) {
          if (row[idField] == idVal) index = idx;
        })
        if (index >= 0) rCks.splice(index, 1);
      }
      var rIdx = leftTb.datagrid('getRowIndex', idVal);
      if (rIdx >= 0) leftTb.datagrid('uncheckRow', rIdx);

      if (opts.limit && ckList.length < opts.limit) {
        $('.datagrid-cell-check,.datagrid-header-check', leftTb.parent()).removeClass('limited');
      }
      
      _updateCount(ctn);
}
function _updateCount(ctn){
	 var tb = $('.couple-right-table', ctn),
       tbData =  tb.datagrid('getData'),
       rows = tbData.originalRows||[],
	 	   countDiv = $('.selected-count',ctn);
	 if(isLocalZH) countDiv.html('共'+rows.length+'记录');
	 else countDiv.html('Total '+rows.length);
   $('.count-span',ctn).text(rows.length);
}
(function($) {
  function _init(ctn, options) {
    var ctnId = $(ctn).attr('id') || '',
      divprefix = $('<div class="couple-prefix"></div>'),
      divleft = $('<div class="couple-left"></div>'),
      divright = $('<div class="couple-right"></div>'),
      tbleft = $('<table class="couple-left-table"></table>'),
      tbright = $('<table class="couple-right-table"></table>'),
      isparsed = false,
      oldOpts = $(ctn).data('opts');
    if (oldOpts) isparsed = true;

    if (isparsed) {
      options = $.extend(oldOpts, options);
      divleft = $('div.couple-left', ctn);
      divright = $('div.couple-right', ctn);
      tbleft = $('table.couple-left-table', ctn);
      tbright = $('table.couple-right-table', ctn);
    } else {
      var selectedMsg = $.fn.pairgrid.defaults.selectedMsg,
          countDom = $('<div style="display: flex; font-size: 12px; align-items: center; position: absolute;right: 0px;z-index: 100;top: -16px;">'+selectedMsg+' (<span class="count-span" style="color:#4d84ff;"></span>)</div>'),
          countBt = $('<div class="el-icon el-icon-circle-down" style="font-size: 12px;margin-left: 5px;"></div>');
      $(ctn).append(countDom);
      countDom.append(countBt);
      countBt.on('click',function(){
        if(countBt.hasClass('el-icon-circle-down')) {
          countBt.addClass('el-icon-circle-up').removeClass('el-icon-circle-down');
        }else{
          countBt.addClass('el-icon-circle-down').removeClass('el-icon-circle-up');
        }
        divright.toggleClass('show');
      })

      oldOpts = options;
      if(options.prefix) {
    	  divprefix.append($(options.prefix));
    	  $(ctn).append(divprefix);
      }
      
      var totalDiv = $('<div class="selected-count" style="display: none;height: 15px;padding: 8px 20px;text-align: right;border: 1px solid #D1ECF5;border-top-width: 0px;">共0记录</div>');
      
      $(ctn).append(divleft.append(tbleft)).append(divright.append(tbright).append(totalDiv));
    }
    $(ctn).data('opts', oldOpts);
    if (ctnId) {
      tbleft.attr('id', ctnId + '_left');
      tbright.attr('id', ctnId + '_right');
    }
    if (options.zone) {
      var zarr = options.zone;
      if (zarr[0] && zarr[1]) {
        divleft.css({
          flex: '1 1 ' + zarr[0] + '%'
        });
        divright.css({
          flex: '1 1 ' + zarr[1] + '%'
        });
      }
    }
    /* options init */
    var basic = {
        checkOnSelect: false,
        selectOnCheck: false,
        idField: options.idField,
        fit: true,
        singleSelect: true,
        fitColumns: true
      },
      colLeft = [],
      colRight = [];

    tbright.data('checklist', []);
    tbleft.data('target', tbright);
    tbright.data('target', tbleft);
    if (options.leftColumns) colLeft = options.leftColumns;
    if (options.rightColumns) colRight = options.rightColumns;
    
    if (options.readonly==true) {
    	ctn.classList.add('readonly');
    }else{
        ctn.classList.remove('readonly');
        colRight.push({
            field: 'rightOperates',
            title: '',
            width: 0,
            _idField: options.idField,
            validEdit: options.validEdit,
            formatter: _rightOpFormatter
        });
    }
    /* right query bar init */
    var barId = ctnId + '_rightbar',
      queryName = options.queryName || 'name';
    _genBar(barId, queryName, ctn, options);
    /* table init */
    var optL = {
    	  pageSize: options.pageSize,
        pageList: options.pageList,
        limit: options.limit,
        pagination: true,
        onCheck: _leftOncheck,
        onUncheck: _leftOnuncheck,
        onLoadSuccess: _leftOnloadsuccess,
        onCheckAll: _leftOncheckAll,
        onUncheckAll: _leftOnuncheckAll,
        columns: [colLeft],
        toolbar: options.toolBar,
        onBeforeLoad: options.leftBeforeLoad,
        onBeforeCheck: function(index, row) {
        	if(row){
                if (row.disabled == true) return false;
                var cklist = tbright.data('checklist');
                if (options.limit && cklist.length >= options.limit) {
                  if (!cklist.includes(row[options.idField])) return false;
                }
        	}
        },
        onBeforeUncheck: function(index, row){
	  	      var tb = $(this),
	  	  	  	  ctn = tb.parents('.couple-wrap');
	  	  	  if(_checkUnselectable(ctn, [row])) return false;
	  	  	 
        }
      },
      optR = {
    	rownumbers: true,
        singleSelect: true,
        onLoadSuccess: _rightOnloadsuccess,
        columns: [colRight],
        toolbar: '#' + barId,
        onBeforeLoad: options.rightBeforeLoad,
        loadFilter: partPurchasePagerFilter,
        pagination: true
      };
    /* url or data */
    if (options.leftUrl) {
      optL.url = options.leftUrl;
    } else if (options.leftData) optL.data = options.leftData;

    if (options.rightUrl) {
      optR.url = options.rightUrl;
    } else if (options.rightData) optR.data = options.rightData;

    tbleft.datagrid($.extend({}, basic, optL));
    tbright.datagrid($.extend({}, basic, optR));

    var pager = tbleft.datagrid('getPager');
    pager.pagination({
        displayMsg: 'Total {total}'
    });
    
  }

  /* create right querybar */
  function _genBar(id, name, ctn, opts) {
    if ($('#' + id, ctn).length > 0) {
    	if(opts.readonly) $('.clear-all', $bar).hide();
        else $('.clear-all', $bar).show();
    	return;
    }
    var messages = opts.messages;
    var queryName = messages.queryName;
    var bar = [];
    bar.push('<div id="' + id + '" class="query_bar" >');
    bar.push('<div class="query-right searchPosition">');
    bar.push('<input name="' + name + '" placeholder="' + queryName + '">');
    bar.push('<b class="el-icon el-icon-common-search"></b>');
    
    bar.push('</div>');
    bar.push('<span class="clear-all el-icon el-icon-operation-delete"></span>');
    bar.push('</div>');

    var $bar = $(bar.join(' ')),
    	clearDom = $('.clear-all', $bar);
    $(ctn).after($bar);
    $('b', $bar).on('click', function(e) {
      var text = $('input', $bar).val() || '',
        tbright = $('.couple-right-table', ctn),
        divright = $('.couple-right', ctn),
        rows = tbright.datagrid('getRows');
      // 新查询逻辑
      searchFun(tbright,name,text);
      return;

      rows.map(function(item, idx) {
        var trStr = 'tr[datagrid-row-index="' + idx + '"]',
          tr = $('.datagrid-view2 ' + trStr, divright),
          trL = $('.datagrid-view1 ' + trStr, divright),
          isMatch = false;
        name.split(',').map(function(nItem) {
          if (text && item[nItem] && item[nItem].indexOf(text.trim()) < 0) {
        	  
          } else if(item[nItem]){
            isMatch = true
          }
        });
        if (isMatch) {
          tr.show();
          trL.show();
        } else {
          tr.hide();
          trL.hide();
        }
      });
    });
    clearDom.on('click', function(e) {
      var tips = messages.tips,
        msg = messages.delete,
        unclearMsg = messages.unclearAll,
        tbright = $('.couple-right-table', ctn),
        rows = tbright.datagrid('getRows');;
      // check immutable items
      if(_checkUnselectable(ctn, rows)) {
    	  $.messager.alert(tips, unclearMsg).addClass("butified");
    	  return false
      }
      
      $.messager.confirm(tips, msg, function(r) {
        if (r) $(ctn).pairgrid('clear');
      }).addClass("seriousConfirm");
    });
    if(opts.readonly) clearDom.hide();
    else clearDom.show();
  }
  // 防抖
  function debounce(fn,delay) {
    return function(args) {
      var that = this,
          _args = args;
      if(fn.id !== null) clearTimeout(fn.id);
      
      fn.id = setTimeout(function(){
        fn.call(that,_args);
      },delay);
    }
  }
  
  function updateRightTotal(right) {
    var pager = right.datagrid('getPager'),
        pagerOps = pager.pagination('options');

    pager.pagination('select',pagerOps.page);
  }
  function reloadRightClick(ctn){
    $('.couple-right .pagination-load',ctn).click();
  }
  /* check unselectable */
  function _checkUnselectable(ctn,rows){
	  var opts = $(ctn).data('opts'),
	  	  has = false;

	  if(opts.validEdit && typeof opts.validEdit == 'function') {
		 rows.map(function(item, idx){
		  	if(opts.validEdit(rows[idx]) === false) {
		  		has = true;
		  	}
		 })
	  }
	  return has;
  }
  /* table event functions */
  function _leftOncheck(index, row) {
    var left = $(this),
      right = left.data('target'),
      opts = left.datagrid('options'),
      ctn = left.parents('.couple-wrap'),
      ctnOpts = ctn.data('opts'),
      idField = opts.idField;
    //var idx = right.datagrid('getRowIndex', row[idField]);
    //if (idx < 0) right.datagrid('appendRow', row);
    var ckList = right.data('checklist');
    if (!ckList.includes(row[idField])) {
      var rightData = right.datagrid('getData'),
          pager = right.datagrid('getPager'),
          pagerOps = pager.pagination('options');

      ckList.push(row[idField]);

      if(rightData.rows.length<pagerOps.pageSize) {
        right.datagrid('appendRow', row);
      }
      pagerOps.total += 1;
      debounce(updateRightTotal,20)(right);

      // 全局数据添加该项
      var originalRows = rightData.originalRows;
      originalRows.push(row);
    }
    if (opts.limit && ckList.length >= opts.limit) {
      $('.datagrid-cell-check,.datagrid-header-check', left.parent()).addClass('limited');
    }
    if(ctnOpts.onCheck) ctnOpts.onCheck.call(this,index, row);
    
    _updateCount(ctn);
  }

  function _leftOnuncheck(index, row) {
    var left = $(this),
      right = left.data('target'),
      opts = left.datagrid('options'),
      idField = opts.idField;
    var idx = right.datagrid('getRowIndex', row[idField]);
    if (idx >= 0) right.datagrid('deleteRow', idx);
    var ckList = right.data('checklist');
    if (ckList.includes(row[idField])) {
      ckList.splice(ckList.indexOf(row[idField]), 1);
      // 全局数据移除该项
      var rtbData = right.datagrid('getData'),
          originalRows = rtbData.originalRows,
          rowKeys = originalRows.map(function(item){ return item[idField]; });
      originalRows.splice(rowKeys.indexOf(row[idField]), 1);
      rtbData.total = rtbData.total - 1;
    }

    if (opts.limit && ckList.length < opts.limit) {
      $('.datagrid-cell-check,.datagrid-header-check', left.parent()).removeClass('limited');
    }
    _updateCount(left.parents('.couple-wrap'));

    var ctn = right.parents('.couple-wrap');
    debounce(reloadRightClick,0)(ctn);
  }

  function _leftOncheckAll(rows) {
    var tb = $(this),
      opts = tb.datagrid('options'),
      right = tb.data('target');
    rows.map(function(item, idx) {
      if (item.disabled == true) tb.datagrid('uncheckRow', idx);
      else tb.datagrid('checkRow', idx);

      var ckList = right.data('checklist');
      if (opts.limit && ckList.length >= opts.limit && !ckList.includes(
          item[opts.idField])) {
        var trStr = 'tr[datagrid-row-index="' + idx + '"]',
          tr = $('.datagrid-view2 ' + trStr, tb.parent()),
          ckipt = $('.datagrid-cell-check input', tr)[0];
        if (ckipt) ckipt.checked = false;
      }
    });

    var hck = tb.parent().find('.datagrid-header-check input')[0];
    if (hck) hck.checked = true;
    
    _updateCount(tb.parents('.couple-wrap'));
  }

  function _leftOnuncheckAll(rows) {
	  var tb = $(this),
	  	  ctn = tb.parents('.couple-wrap');
	  
	  rows.map(function(item, idx) {
		  if(_checkUnselectable(ctn, [item])) tb.datagrid('checkRow', idx);
		  else tb.datagrid('uncheckRow', idx);
	  });
	  _updateCount(ctn);
  }

  function _leftOnloadsuccess(data) {
    $(this).datagrid("enableContextmenuAutoSize");
      var tb = $(this).data('status', 'finish'),
        rightTb = tb.data('target'),
        status = rightTb.data('status'),
        ckList = rightTb.data('checklist');

      if (status == 'finish') { /* init selected */
        /* 结果集映射选择列表 */
        ckList.map(function(idVal) {
          var idx = tb.datagrid('getRowIndex', idVal);
          if (idx >= 0) tb.datagrid('checkRow', idx);
        });
        $(window).resize();
      }
      if(data && data.rows){
        data.rows.map(function(item,idx){
          if(item.disabled==true){
            var left = tb.parents('.couple-left'),
              trStr = 'tr[datagrid-row-index="' + idx + '"]',
              tr = $('.datagrid-view2 ' + trStr, left);
            tr.find('td').addClass('readonly');
          }
    	});
    }
    tb.parents('.couple-wrap').pairgrid('validSelect');
    
    _updateCount(tb.parents('.couple-wrap'));
  }

  function _rightOnloadsuccess(data) {
	  $(this).datagrid("enableContextmenuAutoSize");
    var tb = $(this),
      leftTb = tb.data('target'),
      opts = $(this).datagrid('options'),
      idField = opts.idField || '',
      ckList = $(this).data('checklist');
    if (data && data.originalRows) { /* push selections */
      data.originalRows.map(function(row) {
        if (!ckList.includes(row[idField])) ckList.push(row[idField]);
      });
    }
    
    if($(this).data('status') != 'finish') {
    	$(this).data('originalCheckList',ckList.map(function(idVal) { return idVal; }));
    }
    tb.data('status', 'finish');
    
    var status = leftTb.data('status');
    if (status == 'finish') { /* init selected */
      ckList.map(function(idVal) {
        var idx = leftTb.datagrid('getRowIndex', idVal);
        if (idx >= 0) leftTb.datagrid('checkRow', idx);
      });
      $(window).resize();
    }
    
    _updateCount(leftTb.parents('.couple-wrap'));
  }

  function _rightOpFormatter(value, row, index) {
    var idField = this._idField;
    var div =
      '<div class="pairgrid-op pairgrid-op-remove" onclick="_rightRowClick(this,&quot;' +
      row[idField] + '&quot;,&quot;' + idField + '&quot;)"></div>';
    if(this.validEdit && typeof this.validEdit == 'function') {
    	if(this.validEdit(row) === false) return '';
    }
    return div;
  }

  $.fn.pairgrid = function(options, param) {
    if (typeof options == 'string') {
      return $.fn.pairgrid.methods[options](this, param);
    }
    /* 初始化组件 */
    $(this).addClass('couple-wrap');
    return this.each(function() {
      var parsedOpts = $.fn.pairgrid.parseOptions(this),
        messages = $.extend({}, $.fn.pairgrid.defaults.messages, (parsedOpts.messages || {}),(options.messages||{}));
      var op = $.extend({}, parsedOpts, options, {
        messages: messages
      });
      $.data(this, 'pairgrid', {
        options: op,
        data: []
      });
      _init(this, op);
    });
  }

  $.fn.pairgrid.parseOptions = function(target) {
    var t = $(target);
    return $.extend({}, $.fn.pairgrid.defaults, $.parser.parseOptions(
      target, ['id', 'leftColumns',
        'rightColumns', {
          readonly: 'boolean'
        }
      ]), {
      disabled: (t.attr('disabled') ? true : undefined),
      text: ($.trim(t.html()) || undefined)
    });
  }

  $.fn.pairgrid.methods = {
    getData: function(ctn, params) {
      return $('.couple-right-table', ctn).datagrid('getData').originalRows;
    },
    reload: function(ctn, params) {
      return ctn.each(function() {
        var leftTb = $('.couple-left-table', ctn);
        leftTb.datagrid('reload', params);
      });
    },
    reloadRight: function(ctn, params) {
        return ctn.each(function() {
          var rightTb = $('.couple-right-table', ctn);
          rightTb.datagrid('reload', params);
        });
    },
    load: function(ctn, params) {
      return ctn.each(function() {
        var leftTb = $('.couple-left-table', ctn);
        leftTb.datagrid('load', params);
      });
    },
    loadRight(ctn,data){
    	 var right = $('.couple-right-table', ctn);
    	 right.datagrid('loadData',data);
    },
    clear: function(ctn, params) {
      return ctn.each(function() {
        var right = $('.couple-right-table', ctn),
          left = right.data('target'),
          idField = right.datagrid('options').idField,
          rtbData = right.datagrid('getData'),
          rows = rtbData.rows;
        if (rows.length) {
          for (var i = rows.length - 1; i >= 0; i--) {
            right.datagrid('deleteRow', i);
          }
        }
        left.datagrid('clearChecked');
        // 置空所有内存分页数据
        right.data('checklist', []);
        rtbData.originalRows = [];
        rtbData.total = 0;
        $('.couple-right .pagination-load',ctn).click();
      });
      
      _updateCount(ctn);
    },
    validSelect: function(ctn, params){
    	return ctn.each(function() {
            var opts = ctn.data('opts'),
            	leftTb = $('.couple-left-table', ctn),
            	rows = leftTb.datagrid('getRows');
            
            $('.datagrid-cell-check', leftTb.parent()).each(function(idx,item){
            	if(opts.validEdit && typeof opts.validEdit == 'function') {
            		if(opts.validEdit(rows[idx]) === false) {
            			$(item).addClass('uneditable');
            		}else {
            			$(item).removeClass('uneditable');
            		}
            	}
            })
        });
    },
    clearByIds: function(ctn, ids){
    	return ctn.each(function() {
            var opts = ctn.data('opts'),
            	idField = opts.idField,
            	rightTb = $('.couple-right-table', ctn),
            	leftTb = $('.couple-left-table', ctn),
            	rows = rightTb.datagrid('getRows').map(function(item){return item;});
            
            $('[field=rightOperates]:not(:first)', rightTb.parent()).each(function(idx,item){
            	var idVal = rows[idx][idField];
            	if(ids && idField && ids.includes(idVal)) {
            	      var rIdx = leftTb.datagrid('getRowIndex', idVal);
            	      if (rIdx >= 0) {
            	    	  leftTb.datagrid('uncheckRow', rIdx);
            	      }else {
            	    	  var ckList = rightTb.data('checklist');
                	      rightTb.datagrid('deleteRow', rightTb.datagrid('getRowIndex', idVal));
                	      ckList.splice(ckList.indexOf(idVal), 1);
                          rightTb.data('checklist',ckList);
            	      }

            	}
            });
            
            _updateCount(ctn);
        });
    },
    resize: function(ctn, params){
    	return ctn.each(function() {
            var leftTb = $('.couple-left-table', ctn),
            	rightTb = $('.couple-right-table', ctn);
            leftTb.datagrid('resize');
            rightTb.datagrid('resize');
        });
    }
  }

  $.fn.pairgrid.defaults = {
	  pageSize: 50,
    pageList: [50, 100, 200],
    messages: {
      tips: TiShi,
      delete: QueRenYiChu,
      queryName: 'name',
      unclearAll: 'Including immutable items,unable to clear all.'
    },
    selectedMsg: 'Selected'
  }
})(jQuery)
