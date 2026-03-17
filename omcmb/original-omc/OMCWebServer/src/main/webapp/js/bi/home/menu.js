(function($){
	function _init(ctn,op){
        ctn.innerHTML = '';
        ctn.classList.add('cmenu');
        _initEvent(ctn,op);/*绑定事件*/
        if(op.url){
        	$.getJson(op.url,function(data){
        		_createMenuItem(ctn,data);
        	});
        }else if(op.data) _createMenuItem(ctn,op.data,op);
    }
    function clearPX(str) {
    	return str.replace('px','')*1;
    }
    function _createMenuItem(pNode,data,op){/* 迭代生成子孙节点 */
    	data.map(function(item){
    		var itemNode = $('<div class="cmenu-item"></div>'),
            	spanNode = $('<span>'+item.text+'</span>');
        
    		if(item.cls) itemNode.addClass(item.cls);
    		itemNode.data('info',item);
    		if(item.show == false) itemNode.addClass('menu-hidden');
    		if(item.disable == true) itemNode.addClass('disabled');
    		$(pNode).append(itemNode.append(spanNode));

    		if(item.children) {
    			var childNode = $('<div class="item-child"></div>');
    			$(itemNode).append(childNode);
    			itemNode.off('mouseenter').on('mouseenter',function(event){//子级菜单位置动态调整
    				var cHeight = $('> .cmenu-item:not(.menu-hidden)',childNode).length*36;// 子菜单高度计算：可见的子菜单个数 * 高度
    				var distance = cHeight + event.clientY - document.documentElement.clientHeight;
    				if(distance > 0) {
    					var style = getComputedStyle(this),
        		  	  		itemH = clearPX(style.height) + clearPX(style.paddingTop) + clearPX(style.paddingBottom) 
        		  	  		  		+ clearPX(style.borderTopWidth) + clearPX(style.borderBottomWidth),
        		  	  		times = (distance - distance % itemH)/itemH + 1;
    					childNode.css({top: -itemH*times+'px'});
    				}else childNode.css({top: 0});
    			});
    			_createMenuItem(childNode,item.children,op);
    		}
    	});
    }
    function _initEvent(ctn,op){
    	$(ctn).off('click');
    	$(ctn).on('click',function(event){
    		//var event = arguments.callee.caller.arguments[0] || window.event;
    		var node = event.target, cList = node.classList;
    		if(!Array.from(cList).includes('cmenu-item')) node = $(node).parents('.cmenu-item:first');
    		var infoRow = $(node).data('info');
    		if(!infoRow.disable && op.click && typeof op.click == 'function') op.click(infoRow);
    		$('.item-child').addClass('menu-hidden');
    		event.stopPropagation();
    		setTimeout(function(){ $('.item-child').removeClass('menu-hidden'); },50);
    	});
    	$(ctn).off('mouseenter').on('mouseenter',function(){
    		var pageX =  document.documentElement.clientWidth,
    			itemX = ctn.offsetLeft,
    			isLess = pageX-itemX-100<200;
    		if(isLess){
    			ctn.classList.remove('item-left');
    			ctn.classList.add('item-right');
    		}else{
    			//ctn.classList.add('item-left');
    			//ctn.classList.remove('item-right');
    			$(ctn).addClass('item-left');
    			$(ctn).removeClass('item-right')
    		}
    	});
    }
    $.fn.cmenu = function(options,param){
    	/* 判断是否为对外调用API */
    	if(typeof option == 'string') $.fn.cmenu.methods[options](this,param);
    	/* 初始化组件 */
    	var op = $.extend({},$.fn.cmenu.defaults,options);
    	return this.each(function(){
    		_init(this,op);
    	});
    }
    $.fn.cmenu.methods = {

    }
    $.fn.cmenu.defaults = {

    }
})(jQuery)