<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>
<style>
.EPCInfo{
	position:absolute;
	top:30px;
	bottom:0px;
	width:100%;
}
.newEGW , .editEpc{
	height:100%;
	top:0;
	position:absolute;
	display:none;
	z-index:10;
	background:white;
	padding:20px 60px 0px;	
}
.eGWBasicInfoItemDiv{
	display:inline-block;
	width:400px;
	margin:5px 40px 0 0;
	vertical-align:top;
}
.eGWBasicInfoItemDiv label{
	display:block;
	line-height:25px;
	color:#797979;
}
.eGWBasicInfoItemDiv input{
	width:350px;
}
.eGWBasicInfoItemDiv img{
	margin-left:10px;
}
.eGWBasicInfoItemDiv .prompt{
	display:block;
	line-height:25px;
	height:25px;
	color: red;
}
.submitConfig{
	margin-top:15px;
}
.tabsTitle{
	border:none;
}
.closeButton{
	display:none
}
</style>

<div class="panelDefault">
	<!-- 右上角添加按钮 -->
	<div class="circleIcon">
		<span class="el-icon el-icon-circle-add addEPC addButton" showOpenType="1" myOpType="1" onclick="addOrCloseEPCPage()"></span>
	
		<span class="el-icon el-icon-circle-close addEPC closeButton" showOpenType="2" myOpType="2" onclick="addOrCloseEPCPage()"></span>
			<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
	</div>

	<!-- EPC列表 -->
	<div class="tabsTitle" style='height:28px;line-height:28px;'>
		<span class="active"><%=rb.getString("ZhuCe")%></span>	
	</div>
	<div class="EPCInfo">
		<table class="easyui-datagrid" id="epcRegisterServerList" fit="true" data-options="border:false,fitColumns:true,singleSelect:true,
                   rownumbers:true,toolbar:'#toolbar_epcRegisterServerList',url:'${ctx}/epc/configuration/getEpcServers.action',pageSize:${pageSize},pageList:${pageList},striped:true,
                   pagination:true,pagePosition:'bottom',idField:'IP',onLoadSuccess: datagridLoadSuccess,onBeforeLoad:beforeload_epcRegister,
                   onLoadError:datagridLoadError">
			<thead>
			<tr>
			    <th data-options="field:'EPC_ID'" hidden="true"></th>
			    <%-- <th data-options="field:'ADDR_TYPE',formatter: operFormatterAddrType" width="50"><%=rb.getString("EPCDiZhiLeiXing")%></th> --%>
				<th data-options="field:'NAME'" width="50"><%=rb.getString("EPCMingChen")%></th>
				<th data-options="field:'IP'" width="50"><%=rb.getString("IPDiZhi")%></th>
				<th data-options="field:'PORT'" width="50"><%=rb.getString("DuanKou")%></th>
				<th data-options="field:'operation',formatter: operFormatter,fixed:true" width="80"><%=rb.getString("CaoZuo")%></th>
				<th data-options="field:'CREATE_TIME'" hidden="true"></th>
			</tr>
			</thead>
		</table>
	</div>
	
	<!-- 注册EPC界面 -->
	<div class="newEGW">
		<div class="NeweGWTab1">
			<div class="eGWBasicInfo">
				<div class="eGWBasicInfoItemDiv">
					<label for="EpcName"><%=rb.getString("EPCMingChen")%></label>
					<input id="EpcName" type="text" value=""  class="easyui-validatebox border border-box" style="width:360px;" onblur="checkRangLength(this.id,this.value,1,30)" />
					<span id="EpcNameCheckSpan" class="prompt" ></span>
			    </div>
			    <div class="eGWBasicInfoItemDiv">
					<label for="EpcIP"><%=rb.getString("IPDiZhi")%></label>
					<input id="EpcIP" type="text" value=""  class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpAddressFormat(this.id,this.value)"/>
					<span id="EpcIPCheckSpan" class="prompt" ></span>
			    </div>
			    <div class="eGWBasicInfoItemDiv">
					<label for="EpcPort"><%=rb.getString("DuanKou")%></label>
					<input id="EpcPort" type="text" value=""  class="easyui-validatebox border border-box" style="width:360px;" onblur="checkPort(this.id,this.value)"/>
					<span id="EpcPortCheckSpan" class="prompt" ></span>
			    </div>
			    <div class="eGWBasicInfoItemDiv">
			    	<input id="EpcAddrType" type="checkbox" value="" style="display:block;width:initial;"/>
					<label for="EpcAddrType">Signalling</label>
					
			    </div>
			</div>
			<div class="submitConfig">
		    	<a href="#" class="linkbutton" style="float:left;" onclick="addEpcServerAddr()"><span><%=rb.getString("BaoCun")%></span></a>
		  	</div>
		</div>
	</div>
	
	<!-- 修改EPC界面 -->
	<div class="editEpc">
		<div class="NeweGWTab1">
			<div class="eGWBasicInfo">
			   <%--  <div class="eGWBasicInfoItemDiv">
					<label for="modifyEpcAddrType"><%=rb.getString("EPCDiZhiLeiXing")%></label>
                       <select id="modifyEpcAddrType" class="easyui-combobox border border-box" style="height:27px;width:360px;" data-options="editable:false" disabled="disabled">
                         <option value="1">Configured</option>
                         <option value="2">Signalling</option>
                       </select>
					<span id="modifyEpcAddrTypeCheckSpan" class="prompt" ></span>
			    </div> --%>
				<div class="eGWBasicInfoItemDiv">
					<label for="modifyEpcName"><%=rb.getString("EPCMingChen")%></label>
					<input id="modifyEpcName" type="text" value=""  class="easyui-validatebox border border-box" style="width:360px;" onblur="checkRangLength(this.id,this.value,1,30)" />
					<span id="modifyEpcNameCheckSpan" class="prompt" ></span>
			    </div>
			    <div class="eGWBasicInfoItemDiv">
					<label for="modifyEpcIP"><%=rb.getString("IPDiZhi")%></label>
					<input id="modifyEpcIP" type="text" value=""  class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpAddressFormat(this.id,this.value)"/>
					<span id="modifyEpcIPCheckSpan" class="prompt" ></span>
			    </div>
			    <div class="eGWBasicInfoItemDiv">
					<label for="modifyEpcPort"><%=rb.getString("DuanKou")%></label>
					<input id="modifyEpcPort" type="text" value=""  class="easyui-validatebox border border-box" style="width:360px;" onblur="checkPort(this.id,this.value)"/>
					<span id="modifyEpcPortCheckSpan" class="prompt" ></span>
			    </div>
			    <div class="eGWBasicInfoItemDiv">
			    	<input id="modifyEpcAddrType" type="checkbox" value="" style="display:block;width:initial;"/>
					<label for="modifyEpcAddrType">Signalling</label>
			    </div>
			</div>
			<div class="submitConfig">
		    	<a href="#" class="linkbutton" style="float:left;" onclick="updateEpcServerAddr()"><span><%=rb.getString("BaoCun")%></span></a>
		 	</div>
		</div>
	</div>
</div>

<%-- 工具栏 - EPC注册--%>
<div id="toolbar_epcRegisterServerList" class="toolbarContainer">
	<div class="queryGroup">
		<input id="EPCMolnitorSearch" name="task_name" placeholder="<%=rb.getString("EPCMingChen")%>&nbsp;/&nbsp;<%=rb.getString("IPDiZhi")%>" />
		<b class="el-icon el-icon-common-search" onclick="$('#epcRegisterServerList').datagrid('reload')"></b>
	</div>
</div>

<script>
    $(function(){
    	$('#EPCMolnitorSearch').bind('keyup',function(e){
    		if(e.keyCode == 13){
    			$('#epcRegisterServerList').datagrid('reload')
    		}
    	})
    });
	/* 初始化  -- 表格操作列  */
	function operFormatter(value, rowData, rowIndex) {
		var res = "";
		res = "<div class='el-icon el-icon-operation-edit' style='margin-left: 15px' title='modify' onclick='modifyEpcServer(\""
				+ rowIndex + "\")'></div>";
		res += "<div class='el-icon el-icon-operation-delete' style='margin-left: 15px' title='delete' onclick='deleteEpcServer(\""
			    + rowIndex + "\")'></div>";
		return res;
	}
	
	/* 修改EPC */
	function modifyEpcServer(idx){
		
		//初始化数据 
		var row=$("#epcRegisterServerList").datagrid('getData').rows[idx];
		
		if(row.ADDR_TYPE == 2){
			$("#modifyEpcAddrType").prop("checked",true);
		}else{
			$("#modifyEpcAddrType").prop("checked",false);
		}
		$("#modifyEpcAddrType").prop("disabled",true);
		$("#modifyEpcName").val(row.NAME);
		$("#modifyEpcIP").val(row.IP);
		$("#modifyEpcPort").val(row.PORT);
		
		$(".editEpc").slideDown(200); 
		$(".addEPC").attr("myOpType",2);
		$(".circleBg").removeClass("add_circle");
		$(".circleBg").addClass("close_circle");
		$(".titleButtonText").text("<%=rb.getString("GuanBi")%>");
	}
	
	/* 删除EPC */
	function deleteEpcServer(idx){
		var row=$("#epcRegisterServerList").datagrid('getData').rows[idx];
		
		if(isEpcTraceByEpcId(row.EPC_ID)){
			showMsg('prompt_msg',"<%=rb.getString("EPCFuWuBeiZhanYongBuKeShanChu")%>");
			return;
		}
		
		var req={};
		req["NAME"]=row.NAME;
		req["IP"]=row.IP;
		req["PORT"]=row.PORT;
		req["CREATE_TIME"]=row.CREATE_TIME;
		req["EPC_ID"]=row.EPC_ID;
		req["ADDR_TYPE"]=row.ADDR_TYPE;
		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuRenWu")%>", function(r) {
			if(r){	
				$.post("${ctx}/epc/configuration/deleteEpcServerByIP.action",req,function(data){
			          if (data["success"]) {
			        	  showMsg('success_msg',"<%=rb.getString("ShanChuChengGong")%>");
			        	  $("#epcRegisterServerList").datagrid("reload");
			          } else {
			        	  showMsg('error_msg',data["message"]);
			          }
			      }, "json");
			 }
	    }).addClass("seriousConfirm");
	}
	
	/* EPC id校验 */
	function isEpcTraceByEpcId(EPCID){
		var flag=false;
		$.ajax({
			type: "post",
			url: "${ctx}/epc/configuration/isEpcTraceByEpcId.action",
			data: {"EPC_ID":EPCID},
			async: false,
			dataType:"json",
			success: function(data) {
				flag=data["success"];
			}
		});
		return flag;
	}
	
	
	/* 打开或关闭EPC新建/修改界面 */
	function addOrCloseEPCPage(){ //newEGW
		var myOpType = $(".addEPC").attr("myOpType");
		var rBottomNew = $(".newEGW").position().top;
		$(".newEGW .eGWBasicInfoItemDiv input").val("");
		$(".newEGW .prompt").text("");
		
		if(myOpType == 1){
	    	
			$(".circleBg").removeClass("add_circle");
			$(".circleBg").addClass("close_circle");
			$(".newEGW").slideDown(200);
			$(".titleButtonText").text("<%=rb.getString("GuanBi")%>");
			$(".addEPC").attr("myOpType",2);
			$('.addButton').css('display','none')
			$('.closeButton').css('display','block')
		}else{
			
			$(".circleBg").addClass("add_circle");
			$(".circleBg").removeClass("close_circle");
			$(".newEGW .newEGWTit > li").first().click();
			$(".newEGW").slideUp(500);
			$(".editEpc").slideUp(200); 
			$("#EpcAddrType").prop("checked",false);
			$(".titleButtonText").text("<%=rb.getString("TianJia")%>");
			$(".addEPC").attr("myOpType",1);
			$('.addButton').css('display','block')
			$('.closeButton').css('display','none')
		}	
	}

	function addEpcServerAddr(){
		//必填项、格式校验

		if(!checkRangLength("EpcName",$("#EpcName").val(),1,30)){
			return;
		}
		if(!checkIpAddressFormat("EpcIP",$("#EpcIP").val())){
			return;
		}
		if(!checkPort("EpcPort",$("#EpcPort").val())){
			return;
		}
		var params={};
		
		if($("#EpcAddrType").prop("checked")){
			params["ADDR_TYPE"]=2;
		}else{
			params["ADDR_TYPE"]=1;
		}
		params["IP"]=$("#EpcIP").val();
		params["NAME"]=$("#EpcName").val();
		params["PORT"]=$("#EpcPort").val();
		savingCover();
		$.post("${ctx}/epc/configuration/addEpcServerAddr.action", params, function(data){//{epc_addr:JSON.stringify(params)}
			cancelSavingCover();
            if (data["success"]) {
				$(".newEGW").slideUp(500);
				$(".circleBg").addClass("add_circle");
				$(".circleBg").removeClass("close_circle");
				$("#EpcAddrType").prop("checked",false);
				$(".titleButtonText").text("<%=rb.getString("TianJia")%>");
            	showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
            	$("#epcRegisterServerList").datagrid("reload",{});
            } else {
				$(".newEGW").slideUp(500);
            	showMsg('error_msg',data["message"]);
            }
        }, "json"); 
		
	}
	function updateEpcServerAddr(idx){
		
		//校验数据格式
		if(!checkRangLength("modifyEpcName",$("#modifyEpcName").val(),1,30)){
			return;
		}
		if(!checkIpAddressFormat("modifyEpcIP",$("#modifyEpcIP").val())){
			return;
		}
		if(!checkPort("modifyEpcPort",$("#modifyEpcPort").val())){
			return;
		}
		
		var row=$("#epcRegisterServerList").datagrid('getSelected');
		
		if(row.IP!=$("#modifyEpcIP").val() || row.NAME!=$("#modifyEpcName").val() ||row.PORT!=$("#modifyEpcPort").val()){
			var params={};
			params["IP"]=$("#modifyEpcIP").val();
			params["NAME"]=$("#modifyEpcName").val();
			params["PORT"]=$("#modifyEpcPort").val();
			params["OLD_IP"]=row.IP;
			params["EPC_ID"]=row.EPC_ID;
			savingCover();
			$.post("${ctx}/epc/configuration/updateEpcServerAddr.action", params, function(data){
				$(".editEpc").slideUp(200);
				cancelSavingCover();
	            if (data["success"]) {
	            	showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
	            	$("#epcRegisterServerList").datagrid("reload",{});
	            } else {
	            	showMsg('error_msg',data["message"]);
	            }
	        }, "json");
			
		}else{
			//如果没有修改,保存的时候直接关掉窗口
			$(".editEpc").slideUp(200);
		}
		
	}

	function isValidIP(ip){     
	    var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
	    return reg.test(ip);     
	}
	
	function checkIpAddressFormat(id,value){
	    if(!isValidIP(value)){
	    	$("#"+id).focus().select();
	        $("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>");
	        return false;
	    }else{
	    	$("#"+id+"CheckSpan").text("");
	    	return true;
	    }
	}
	
	//校验端口号 范围 0-65535
	function checkPort(id,value){
		if(isNumeric(value)&& parseInt(value)>=0 && parseInt(value)<=65535){
			$("#"+id+"CheckSpan").text("");
			return true;
		}else{
			$("#"+id).focus().select();
	        $("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuYouXiaoDuanKou")%>");
	        return false;
		}
	}
	
	function isNumeric(str) {
	    if(str.length==0){
	    	return false;
	    }
	    for(var i=0;i<str.length;i++){
	    	if(str.charAt(i)<"0" || str.charAt(i)>"9"){
	    		return false;
	    	}
	    }
	    return true;  
	}
	
	//检查长度
	function checkRangLength(id,value,minLength,maxLength){
		if(value=="" || value.length<minLength || value.length>maxLength){
			$("#"+id).focus().select();
			$("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuDeMingChenZai")%> {"+minLength+"} <%=rb.getString("AND")%> {"+maxLength+"} <%=rb.getString("ChangDu")%>");
			return false;
		}else{
			$("#"+id+"CheckSpan").text("");
			return true;
		}
	}
	//查询epc register
	function beforeload_epcRegister(params){
		params["searchText"] = $('#EPCMolnitorSearch').val();
		params["timeZone"] = timeZone;
	}
</script>