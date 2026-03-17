(function($) {
  function _init(ctn, options) {
    var $ctn = $(ctn),
      ctnId = options.id || '', barId = ctnId + '_gridbox_query',
      ctnName = options.name || '', txtName = options.textname || ctnName + '_text',
      isparsed = false, oldOpts = $ctn.data('opts');
    if (oldOpts) isparsed = true;
    $ctn.hide().removeAttr('name').attr('gridboxname', ctnName);

    var span = $('<span class="gridbox-wrap"></span>'),
      input = $('<input class="gridbox-text" name="' + txtName + '"/>'),
      hide = $('<input class="gridbox-hidden" type="hidden" name="' + ctnName + '"/>'),
      tbspan = $('<span class="gridbox-slide"></span>'), tb = $('<table class="gridbox"></table>'),
      $ok = $('<span class="gridbox-ok"></span>'), $Edit = $('<span class="gridbox-edit operation_edit"></span>');
    if (isparsed) { /* 处理重复渲染 */
      options = $.extend(oldOpts, options);
      span = $ctn.next('.gridbox-wrap');
      input = $('.gridbox-text', span);
      hide = $('.gridbox-hidden', span);
      tbspan = $('.gridbox-slide', span);
      tb = $('.gridbox', span);
      $ok = $('.gridbox-ok', span);
      $Edit = $('.gridbox-edit', span);
    } else {
      $ok.text(options.messages.ok);
      tbspan.append(tb).append($ok);
      span.append(input).append(hide).append(tbspan);
      if (!options.readonly) span.append($Edit);
      $ctn.after(span);
      _genBar(barId, options, ctn);
    }
    $ctn.data('opts', options);
    if (options.readonly) span.addClass('readonly');
    if (!options.editable) {
      input.attr('readonly', true);
      $Edit.addClass('unEditable');
    }
    input.val(options.text || options.value || '');
    hide.val(options.value || '');
    span.css({
      width: options.width
    });
    tbspan.css({
      width: options.width < 340 ? 340 : options.width
    });
    /* botton events */
    $Edit.off('click').on('click', function(e) {
      var isVisible = tbspan.is(':visible');
      if (isVisible) tbspan.slideUp();
      else {
    	var rows = tb.datagrid('getRows'),
    		curVal = $ctn.gridbox('getValue'),
    		has = false;
    	rows.map(function(row,idx){
    		if(row[options.valueField] == curVal) {
    			has = true;
    			tb.datagrid('checkRow',idx);
    		}
    	})
    	if(!has) tb.datagrid('clearSelections');
        tbspan.slideDown(500, function() {
          tb.datagrid('resize');
        });
      }
      e.stopPropagation();
    });
    if (options.editable) {
      input.off('click').on('focus', function(e) {
        tbspan.slideUp();
      });
    } else {
      input.off('click').on('focus', function(e) {
        $Edit.click();
      });
    }

    input.off('change').on('change', function(e) {
    	$ctn.gridbox('setItem',{value: this.value});
	});
    $ok.off('click').on('click', function(e) {
      var sRows = tb.datagrid('getSelected');
      if (sRows) { /* 确认设值 */
        var txt = options.formatter(sRows[options.textField], sRows);
        input.val(txt);
        hide.val(sRows[options.valueField] || '');
        tbspan.slideUp();
        if(options.onConfirm) options.onConfirm({text: txt,value: sRows[options.valueField]});
      } else {
        var tips = options.messages.tips,
          msg = options.messages.confirm;
        $.messager.confirm(tips, msg, function(r) {
          if (r) tbspan.slideUp();
          event.stopPropagation();
        }).addClass("seriousConfirm");
      }
    });
    span.off('click').on('click', function(e) {
      e.stopPropagation();
    });
    $(document).on('click', function(e) { /* 非输入域点击收起 */
      tbspan.slideUp();
    });
    options.toolbar = '#' + barId;
    _initgrid(tb, options);
  }
  /* init table */
  function _initgrid(tb, options) {
    var opts = {
        columns: []
      },
      clms = [{
        title: '',
        field: 'ck',
        formatter: _statusFormatter
      }];
    ['data', 'url', 'pagination', 'fitColumns', 'fit', 'singleSelect',
      'toolbar', 'onBeforeLoad', 'queryParams', 'onLoadSuccess', 'idField'
    ].map(function(key) {
      if (options[key]) opts[key] = options[key];
    });
    if (options['columns']) {
      options['columns'].map(function(item) {
        clms.push(item);
      });
    } else {
      clms.push({
        title: 'Name',
        field: options.textField,
        width: 100
      });
    }
    opts['columns'].push(clms);
    if (opts.url) delete opts.data;
    tb.datagrid(opts).datagrid('getPager').pagination({
      displayMsg: ''
    });
  }
  /* create right querybar */
  function _genBar(id, options, ctn) {
	  var name = options.queryName
    if ($('#' + id, ctn).length > 0) return;
    var queryName = options.messages.queryName;
    var bar = [];
    bar.push('<div id="' + id + '" class="query_bar" >');
    bar.push('<div class="query-right searchPosition">');
    bar.push('<input name="' + name + '" placeholder="' + queryName + '">');
    bar.push('<b class="el-icon el-icon-common-search"></b>');
    bar.push('</div>');
    bar.push('</div>');

    var $bar = $(bar.join(' '));
    $(ctn).after($bar);
    $('b', $bar).on('click', function(e) {
      var gridCtn = $(ctn).next('.gridbox-wrap'),
        tb = $('.gridbox', gridCtn),
        inpt = $('#' + id + ' input', gridCtn),
        param = tb.datagrid('options').queryParams||{};
      param[name] = inpt.val();
      tb.datagrid('load', param);
    });
  }

  function _statusFormatter(value, row, index) {
    return '<span class="gridbox-status"></span>'
  }

  $.fn.gridbox = function(options, param) {
    if (typeof options == 'string') {
      return $.fn.gridbox.methods[options](this, param);
    }
    /* 初始化组件 */
    return this.each(function() {
      var op = $.extend({}, $.fn.gridbox.defaults, $.fn.gridbox.parseOptions(
        this), options);
      $.data(this, 'gridbox', {
        options: op
      });
      _init(this, op);
    });
  }

  $.fn.gridbox.parseOptions = function(target) {
    var t = $(target);
    return $.extend({}, $.parser.parseOptions(target, ['id', 'name',
      'textname', 'value', 'text',
      'columns', {
        readonly: 'boolean'
      }
    ]), {
      disabled: (t.attr('disabled') ? true : undefined),
      text: ($.trim(t.html()) || undefined)
    });
  }

  $.fn.gridbox.methods = {
    getValue: function(ctn, params) {
      var hide = $('.gridbox-hidden', $(ctn).next('.gridbox-wrap'));
      return hide.val();
    },
    getText: function(ctn, params) {
      var input = $('.gridbox-text', $(ctn).next('.gridbox-wrap'));
      return input.val();
    },
    setItem: function(ctn,param){
    	return ctn.each(function(idx,item){
    		var input = $('.gridbox-text', $(ctn).next('.gridbox-wrap')),
    			hide = $('.gridbox-hidden', $(ctn).next('.gridbox-wrap'));
    		if(param.value != undefined) hide.val(param.value);
    		if(param.text != undefined) input.val(param.text);
    	});
    }
  }

  $.fn.gridbox.defaults = {
    fit: true,
    fitColumns: true,
    valueField: 'name',
    textField: 'name',
    queryName: 'name',
    width: 340,
    pagination: true,
    singleSelect: true,
    editable: true,
    onConfirm: function(p){},
    formatter: function(text, row) {
      if (text) return text;
      else return '';
    },
    messages: {
      tips: 'tips',
      confirm: 'No selections,sure to close ?',
      ok: 'OK',
      queryName: 'Name'
    }
  }
})(jQuery)
