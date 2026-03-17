// extend the [phone,idCard,date,time,radio,checkbox] rules    
$.extend($.fn.validatebox.defaults.rules, {    
    phone: {    
        validator: function(value,param){
			if(!(/^1[34578]\d{9}$/.test(value))) return false; 
            return true;
        },    
        message: 'Field do not match.'   
    },
	idCard: {    
        validator: function(value,param){
			var reg = /(^[1-9]\d{5}(18|19|([23]\d))\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$)|(^[1-9]\d{5}\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{2}$)/;
			if(reg.test(value)) return true; 
            return false;
        },    
        message: 'Field do not match.'   
    },
	date: {    
        validator: function(value,param){
			var reg = /((^((1[8-9]\d{2})|([2-9]\d{3}))([-\/\._])(10|12|0?[13578])([-\/\._])(3[01]|[12][0-9]|0?[1-9])$)|(^((1[8-9]\d{2})|([2-9]\d{3}))([-\/\._])(11|0?[469])([-\/\._])(30|[12][0-9]|0?[1-9])$)|(^((1[8-9]\d{2})|([2-9]\d{3}))([-\/\._])(0?2)([-\/\._])(2[0-8]|1[0-9]|0?[1-9])$)|(^([2468][048]00)([-\/\._])(0?2)([-\/\._])(29)$)|(^([3579][26]00)([-\/\._])(0?2)([-\/\._])(29)$)|(^([1][89][0][48])([-\/\._])(0?2)([-\/\._])(29)$)|(^([2-9][0-9][0][48])([-\/\._])(0?2)([-\/\._])(29)$)|(^([1][89][2468][048])([-\/\._])(0?2)([-\/\._])(29)$)|(^([2-9][0-9][2468][048])([-\/\._])(0?2)([-\/\._])(29)$)|(^([1][89][13579][26])([-\/\._])(0?2)([-\/\._])(29)$)|(^([2-9][0-9][13579][26])([-\/\._])(0?2)([-\/\._])(29)$))/;
			if(reg.test(value)) return true; 
            return false;
        },    
        message: 'Field do not match.'
    },
	time: {    
        validator: function(value,param){
			var reg = /^((((1[6-9]|[2-9]\d)\d{2})-(0?[13578]|1[02])-(0?[1-9]|[12]\d|3[01]))|(((1[6-9]|[2-9]\d)\d{2})-(0?[13456789]|1[012])-(0?[1-9]|[12]\d|30))|(((1[6-9]|[2-9]\d)\d{2})-0?2-(0?[1-9]|1\d|2[0-8]))|(((1[6-9]|[2-9]\d)(0[48]|[2468][048]|[13579][26])|((16|[2468][048]|[3579][26])00))-0?2-29-)) (20|21|22|23|[0-1]?\d):[0-5]?\d:[0-5]?\d$/;
			if(reg.test(value)){
				return true; 
			} 
            return false;
        },    
        message: 'Field do not match.'   
    },
	radio: {  
		validator: function(value, param){  
			var input = $(param[0]),status = false,firstObj = $(input[0]),
				cntObj = firstObj.parent(),initCount = cntObj.attr("initCount") || 0;
			input.off('.radio').on('click.radio',function(){
				$(this).focus();
				try{ cntObj.tooltip('hide'); }catch(e){}
			});
			cntObj.off("mouseover mouseout").on("mouseover mouseout",function(event){
				var bool = input.validatebox('isValid');
				if(event.type == "mouseover"){
					if(bool) try{ cntObj.tooltip('hide'); }catch(e){}
					else try{ cntObj.tooltip('show');}catch(e){}
				}else if(event.type == "mouseout") try{ cntObj.tooltip('hide'); }catch(e){}
			});
			if(initCount-1<0){
				tipProcess(firstObj,"initCount");
				initCount ++ ;
			}
			status = $(param[0] + ':checked').val() != undefined;
			return status;
		},  
		message: 'Please choose option for {1}.'  
	},
	checkbox: {
		validator: function (value, param) {
			var inputs = $(param[0]), maxNum = param[1], checkNum = 0,status=false,
				firstObj = $(inputs[0]),cntObj = firstObj.parent(),initCount = cntObj.attr("initCount") || 0;
			inputs.each(function () { 
				if (this.checked) checkNum++;
			});
			inputs.off('.checkbox').on('click.checkbox',function(){
				//$(this).focus();
				var bool = inputs.validatebox('isValid');
				if(bool) try{ cntObj.tooltip('hide'); }catch(e){}
				else try{ cntObj.tooltip('show');}catch(e){}
			});
			cntObj.off("mouseover mouseout").on("mouseover mouseout",function(event){
				var bool = inputs.validatebox('isValid');
				if(event.type == "mouseover"){
					if(bool) try{ cntObj.tooltip('hide'); }catch(e){}
					else try{ cntObj.tooltip('show');}catch(e){}
				}else if(event.type == "mouseout") try{ cntObj.tooltip('hide'); }catch(e){}
			});
			if(initCount-1<0){
				tipProcess(firstObj,"initCount");
				initCount ++ ;
			}
			status = checkNum > 0;
			return status;
		},
		message: 'Please choose options !'
	}
});
function tipProcess(firstObj,countFlag){
	var dataOps = firstObj.validatebox('options'),ctn=firstObj.parent(),
		tipMsg = dataOps.missingMessage || dataOps.invalidMessage || firstObj.validatebox.defaults.rules.checkbox.message;
	ctn.tooltip({ position: 'right', content: '<span style="color:#000">'+tipMsg+'</span>',
		onShow: function(){
			$(this).tooltip('tip').css({
				backgroundColor: 'rgb(255, 255, 204)',
				borderColor: 'rgb(204, 153, 51)'
			});
		}
	}).tooltip('hide');
	var initCount = 0;
	if(countFlag) {
		initCount = ctn.attr(countFlag);
		initCount = initCount - 0 + 1;
		ctn.attr(countFlag,initCount);
	}
}