<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>

<style>
.reSetConfirm + .messager-button{
	padding-top:20px;
}
</style>

<ul id="paramNodesUl" class="paramNodesUl">
	<div id="oneMMEPool" style="display: block">
		<li>
			<label for="LTE_SIGLINK_SERVER_LIST_name">${LTE_SIGLINK_SERVER_LIST_name }</label>
			<input id="LTE_SIGLINK_SERVER_LIST_name" name="LTE_SIGLINK_SERVER_LIST" title="${LTE_SIGLINK_SERVER_LIST_title }" class="border border-box" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);validateRTSMMEIP(event);createMML();" js_regex="no_zh"/>
			<div id="LTE_SIGLINK_SERVER_LIST_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
				${LTE_SIGLINK_SERVER_LIST_title }
			</div>
	    </li>
	</div>
	
	<div id="twoMMEPool" style="display: none">
		<li>
			<label for="LTE_X_BAICELLS_POOL_MME_LIST1_name">${LTE_X_BAICELLS_POOL_MME_LIST1_name }</label>
			<input id="LTE_X_BAICELLS_POOL_MME_LIST1_name" name="LTE_X_BAICELLS_POOL_MME_LIST1" title="${LTE_X_BAICELLS_POOL_MME_LIST1_title }" class="border border-box" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);validateRTSMMEIP(event);createMML();" js_regex="no_zh"/>
			<div id="LTE_X_BAICELLS_POOL_MME_LIST1_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
				${LTE_SIGLINK_SERVER_LIST_title }
			</div>
		</li>
		<li>
			<label for="LTE_X_BAICELLS_POOL_MME_LIST2_name">${LTE_X_BAICELLS_POOL_MME_LIST2_name }</label>
			<input id="LTE_X_BAICELLS_POOL_MME_LIST2_name" name="LTE_X_BAICELLS_POOL_MME_LIST2" title="${LTE_X_BAICELLS_POOL_MME_LIST2_title }" class="border border-box" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);validateRTSMMEIP(event);createMML();" js_regex="no_zh"/>
			<div id="LTE_X_BAICELLS_POOL_MME_LIST2_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
				${LTE_SIGLINK_SERVER_LIST_title }
			</div>
		</li>
	</div>
	
</ul>

<script type="text/javascript"> 
	function validateRTSMMEIP(e) {
		var ele = $(e['target']),
			eleId = ele.attr('id'),
			reg = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-4]|2[0-4][0-9]|[01]?[0-9][0-9]?)(,(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-4]|2[0-4][0-9]|[01]?[0-9][0-9]?))*$/,
			val = ele.val().trim(),
			isIP = reg.test(val);
		
		if(val) {
			if(isIP) {
				$('#' + eleId + '_err').removeClass('redColor');
				ele.removeClass('err_border');
			}else {
				$('#' + eleId + '_err').addClass('redColor');
				ele.addClass('err_border');
			}
		}
	}
     //获取当前打开的页面标识
	
	function showMMEPoolWay(flag, row) {
		
		var sels = [];
		var isAllOneMMEPool = true;
		
		if([true, false].includes(flag)) {
			sels = $("#gridCell_cellParam").datagrid("getSelections");
		}
		
		if(row){
			if(flag){
				//新选中
				sels.push(row);
			}else{
				var index = 0;
				for(var i=0; i<sels.length;i++){
					if(sels[i].small_cell_code == row.small_cell_code){
						index = i;
						break;
					}
				}
				sels.splice(index,1);
			}
		}
		
		for (var i = 0; i < sels.length; i++) {
			if (sels[i].mme_enable == "1") {
				isAllOneMMEPool = false;
				break;
			}
		}
		
		if (isAllOneMMEPool && sels.length > 0) {
			
			$("#oneMMEPool").show();
			$("#twoMMEPool").hide();
			
			$("#LTE_SIGLINK_SERVER_LIST_name").val("");
		}else{
			
			$("#oneMMEPool").hide();
			$("#twoMMEPool").show();
			$("#LTE_X_BAICELLS_POOL_MME_LIST1_name").val("");
			$("#LTE_X_BAICELLS_POOL_MME_LIST2_name").val("");
		}
		
	}

    function modMMEPool(sels,mmlstr,hardware_version){
    
    	if(sels.length == 1){
    		doExecute(sels,mmlstr,hardware_version);
    		return;
    	}
    	//当选中的站至少为2个时，触发下面操作
    	if(sels.length >= 2){
    		//选择第一个站的mme_enable作为基准
    		var mmeEnableZero = sels[0].mme_enable;
    		
    		if(mmeEnableZero == null || mmeEnableZero.toString().trim() == ""){
    			mmeEnableZero = 0;
    		}
    		var needPrompt = false;
    		var oneMMEPoolEnb = new Array();
    		
        	for(var i = 0; i < sels.length; i ++){
        		
        		var mmeEnable = sels[i].mme_enable;
        		if("1" != mmeEnable){
        			oneMMEPoolEnb.push(sels[i].serial_number);
        		}
        		
        		if(mmeEnable == null || mmeEnable.toString().trim() == ""){
        			mmeEnable = 0;        		}
        		
        		if(mmeEnable != mmeEnableZero){
        			needPrompt = true;
        		}
        	}
        	
        	if(needPrompt){
        		var sns = "</br>";
        		for(var i = 0;i < oneMMEPoolEnb.length;i ++){
        			sns = sns + oneMMEPoolEnb[i]+",";
        		}
        		
        		sns = sns.substring(0,sns.length-1);
        		
        		$.messager.confirm({
        			width:450,
        			height:220,
        			title:'<%=rb.getString("QueRen")%>',
        			msg:'<%=rb.getString("MMEPoolConfigConfirm")%>' + sns,
        			fn:function(r){
	        			if(r){
	        				doExecute(sels,mmlstr,hardware_version);
	        			}
        			}
        		}).addClass("normalConfirm reSetConfirm");
        	}else {
        		doExecute(sels,mmlstr,hardware_version);
        	}
    	}
    }
    function doExecute(selCells,mmlstr,hardware_version){
    	
    	var smallCells = "";
    	var cellsText = "";
    	var serial_numbers = "";
    	
    	<%--拼接多个小站编码--%>
        for (var codeNum = 0; codeNum < selCells.length; codeNum++) {
    	    smallCells += selCells[codeNum]["small_cell_code"] + ",";
    	    serial_numbers += selCells[codeNum]["serial_number"]+",";
        }

        smallCells = smallCells.substring(0, smallCells.length - 1);
        serial_numbers = serial_numbers.substring(0,serial_numbers.length-1);
        <%--表单序列化--%>
        var paramJson = $('#operValueForm').serializeJson();
        
        <%-- 某个 指标若为空，则表示不对该指标进行设置，从对象中去掉该指标 --%>
        var deleteKeyArr = [];
      
        var param = {};
        param["smallCells"] = smallCells;
        param["serial_numbers"] = serial_numbers;
        param["inputMML"] = encodeURIComponent(mmlstr);
        param["hardwareVersion"] = hardware_version;
        
        updateActionHistoryContent(serial_numbers, "MOD MME",mmlstr);
        
        $.post("${pageContext.request.contextPath}/cell/param/operParamGroupValues.action", param, function (data) {
            if (data["success"]) {
            	closeDefaultWindow();
            } else {
            	
            }
        }, "json");
    	
    }
     
    //进入该页面后执行该方法
	// showMMEPoolWay();
</script>  


