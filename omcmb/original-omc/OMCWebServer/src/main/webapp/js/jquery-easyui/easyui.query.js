(function($){
	// 初始化组件
	function _init(ctn,options){
		var $ctn = $(ctn),
			strList = [
				'<div class="queryGroup">',
			        '<div class="summary-query" style="margin-left:0px;">',
				        '<input style="margin-left:0px;" class="searchInputStyle faultListInput" />',
				       	'<div class="highQueryArrow">',
					       	'<span class="highQueryTip mainCol" style="cursor:pointer;"></span>',
					       	'<span class="el-icon el-icon-common-query-down" flag="1"/>',
				       	'</div>',
				    '</div>',
				    '<div id="cpeisSuper" style="display:inline-block;">',
				        '<b class="el-icon el-icon-common-search"></b>',
				    '</div>',
				'</div>'
		    ];
		var queryDom = $(strList.join(' '));
		if(options.id) {
			queryDom.attr('id',options.id);
		}
		$ctn.addClass('defaultQuery').prepend(queryDom);
		/* 初始化事件 */
		_initEvent(queryDom,options);
	}
	
	function _initEvent(queryDom,options) {
		var arrowDom = $('.highQueryArrow',queryDom),
			tipDom = $('.highQueryTip',queryDom),
			imgDom = $('.el-icon-common-query-down',queryDom),
			searchBt = $('.el-icon-common-search',queryDom),
			sInput = $('.searchInputStyle',queryDom);
		tipDom.text(options.tips);
		$(document).click(function(e){
	        var e = e || window.event;
	        var elem = e.target || e.srcElement;
	        while(elem){
	            if(elem.className == 'operation_more' || elem.className == 'slideDiv'){
	                return
	            }
	            elem = elem.parentNode;
	        }
	        if($(e.target).closest(".window-mask").length==0
	      			 &&$(e.target).closest(".messager-window").length==0 &&$(e.target).closest(".el-icon-common-query-up").length==0
	       			 &&$(e.target).closest(".combo-p").length==0&&$(e.target).closest(".calendar-other-month").length==0
	      			 &&$(e.target).closest(".highQueryTip").length==0&&$(e.target).closest(".query-arrow").length==0
	      			 &&$(e.target).closest("#"+options.targetId).length==0&&$(e.target).closest(".searchInputStyle").length==0){
	           	$("#"+options.targetId).slideUp(300);	
	           	imgDom.attr("flag","1").removeClass('el-icon-common-query-up').addClass('el-icon-common-query-down');
	           	tipDom.hide();
	      	}
	    });
		// 点击高级查询时收缩高级面板
		$("#"+options.targetId).on('click',function(event){
			if($(event.target).closest('.linkbutton_trend').length){
				$(this).slideUp(300);	
	           	imgDom.attr("flag","1").removeClass('el-icon-common-query-up').addClass('el-icon-common-query-down');
	           	tipDom.hide();
			}
		});
	    
		_inputEvent(queryDom,options);
		
		arrowDom.off('click').on('click',function(event){
			_moreQuerySlideFun(queryDom,options);
		});
		
		searchBt.off('click').on('click',function(event){
			if(options.query && typeof options.query == 'function') options.query();
		});
		sInput.off('keyup').on('keyup',function(evt){
			if(evt.keyCode == '13') searchBt.trigger('click');
		})
	}
	// 初始化查询输入域的事件
	function _inputEvent(queryDom,options){
		var inputDom = $('.searchInputStyle',queryDom),
			tipDom = $('.highQueryTip',queryDom),
			imgDom = $('.el-icon-common-query-down',queryDom),
			target = $('#'+options.targetId);
		// 设置输入域的id
		if(options.inputId){
			inputDom.attr('id',options.inputId);
		}
		if(options.name){
			inputDom.attr('name',options.name);
		}
		if(options.value){
			inputDom.val(options.value);
		}
		if(options.placeholder){
			inputDom.prop('placeholder',options.placeholder);
		}
		
		//输入框获得焦点(高级查询)显示
		inputDom.focus(function(){
			//judmentOtherChange("cpeQueryDiv");
			tipDom.show();
		});
		
		imgDom.mouseenter(function(){
			if(inputDom.is(":focus")){
				return;
			} 
			tipDom.show();
		});
		imgDom.mouseleave(function(){
			if(inputDom.is(":focus")){
				return;
			} 
			if(target.is(":visible")){
				return;
			}
			tipDom.hide();
		});

		tipDom.mouseleave(function(){
			if(inputDom.is(":focus")){
				return;
			}
			if(target.is(":visible")){
				return;
			} 
			tipDom.hide();
		});
	}
	
	function _moreQuerySlideFun(div,options){
		var imgDom = $('.el-icon',div),
			target = $('#'+options.targetId);
		if(imgDom.attr("flag")=="1"){
			target.slideDown(500);
			if(options.openAdvance && typeof options.openAdvance == 'function') options.openAdvance();
			imgDom.attr("flag","0").removeClass('el-icon-common-query-down').addClass('el-icon-common-query-up');
		}else{
			target.slideUp(400);
			imgDom.attr("flag","1").removeClass('el-icon-common-query-up').addClass('el-icon-common-query-down');
		}	
	}
	
	$.fn.query = function(options, param) {
	    if (typeof options == 'string') {
	      return $.fn.query.methods[options](this, param);
	    }
	    /* 初始化组件 */
	    return this.each(function() {
	      var parsedOpts = $.fn.query.parseOptions(this),
	      	  tips = parsedOpts.tips?parsedOpts.tips:$.fn.query.defaults.tips;
	      var op = $.extend({}, parsedOpts,{
		    	  tips: tips
		      }, options);
	      
	      _init(this,op);
	    });
	}

	$.fn.query.parseOptions = function(target) {
	    var t = $(target);
	    return $.extend({}, $.fn.query.defaults, $.parser.parseOptions(
	      target, 
	      ['id', 'name','inputId','targetId','placeholder','tips']), {
	      disabled: (t.attr('disabled') ? true : undefined),
	      text: ($.trim(t.html()) || undefined)
	    });
	}
	
	$.fn.query.methods = {
		
	}
	
	$.fn.query.defaults = {
		tips: 'Advanced Query'
	}
})(jQuery)