<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
.egwLabel{
	font-size:12px;
	font-style:normal;
	color:#4C6778;
	display:block;
	margin-bottom:10px;
}
.operationDiv{
	margin-left:5px;
	margin-top:-3px;
}
.middleLine{
	width:75%;
	height:2px;
	background:#DCECF7;
	margin-left:35px;
	margin-bottom:30px;
}
.egwDeviceContent{
	width:75%;
	margin-left:35px;
	display:flex;
}
.egwSelectFileContent{
	width:80%;
	height:415px;
	border:1px solid #CCE1EF;
	margin-left:35px;
}
@media screen and (max-width:1280px){
	.egwDeviceContent{
		width:90%;
	}
	.middleLine{
		width:90%;
	}
	.egwSelectFileContent{
		width:93%;	
	}
}
.upgrade_clear{
	float:right;
	display:inline-block;
	width:20px;
	height:20px;
	margin-right:5px;
	margin-top:8px;
}
.upgrade_delete{
	visibility:hidden;
	transation: visibility 0.5s ease 0.2s;
	cursor:pointer;
	position:absolute;
	top:0px;
	width:20px;
	height:20px;
}
.visible .upgrade_delete {
	visibility:visible;
}
#selectedegwDiv .datagrid-body .datagrid-cell{
	position:relative;
	cursor:pointer;
}
#selectedegwDiv .datagrid-view2 .datagrid-body td{
	cursor:pointer;
}
#selectedegwDiv td{
	border:none;
}
.egwErrorborder{
	border:1px solid #CC0000;
}
.egw_error_mes{
	visibility:hidden;
	color:#CC0000;
	font-size:12px;
}
</style>
<div class='pageDefault slide-position-top'>
	<div class="singleTitle">
		<span><%=rb.getString("XinJianRenWu")%></span>
	</div>
	<div class="slideBody">
		<div class="slideCont">
			<div class="splitGroup">
				<div class="splitGroup_title"><%=rb.getString("JiBenXinXi")%></div>
		  		<div class="splitGroup_body">				
					<div>		
						<label class="inputTittleCss"><%=rb.getString("RenWuMing")%></label>
						<input onblur='egw_checkUpgradeTaskName()' id='egwUpgradeTaskName' value='${taskName }' maxLength='100' style='width:400px;height:26px;margin-bottom:5px;' class="border-box border"/>
						<p class='egw_error_mes'><%=rb.getString("QingShuRuXinJianRenWuMingCheng")%></p>
					</div>
				</div>
			</div>
			<div class="splitGroup">
				<div class="splitGroup_title"><%=rb.getString("SheBeiXuanZe")%></div>
		  		<div class="splitGroup_body">				
					<div style="display: flex;">		
						<div  style='display:inline-block;width:45%;overflow-x:auto;margin-right:30px;float:left '>
							<label class="inputTittleCss"><%=rb.getString("eGWLieBiao")%></label>
							<div  style='height:433px;border:1px solid #E9E9E9;margin-top:10px;'>
								<table id="egwDeviceTable"></table>
							</div>
						</div>
						<div id='selectedegwDiv' style='display:inline-block;overflow-x:auto;width:45%;'>
							<label class="inputTittleCss"><%=rb.getString("YiXuan")%></label>
							<div style='height:433px;border:1px solid #E9E9E9;margin-top:10px;'>
								<table id='selectedegwTable'></table>
							</div>
						</div>
					</div>
					<div id='deviceMesDiv' class='egw_error_mes' style='margin-left:35px;margin-top:6px;'><%=rb.getString("QingXuanZeSheBei")%></div>
				</div>
			</div>
			<div class="splitGroup">
				<div class="splitGroup_title"><%=rb.getString("XuanZeShengJiWenJian")%></div>
		  		<div class="splitGroup_body">				
					<div style="height:300px;">
						<table id='selectUpgradeFileTable'></table>
					</div>
					<div id='selectFileMesDiv' class='egw_error_mes' style='margin-left:35px;margin-top:8px;'><%=rb.getString("QingXianXuanZeWenJian")%></div>
				</div>
			</div>
			<div class="splitGroup">
				<div class="splitGroup_title"><%=rb.getString("XuanZeZhiXingFangShi")%></div>
		  		<div class="splitGroup_body">				
					<div style="margin-top:10px;line-height:26px;">
						<input id='egw_immediate_upgrade' type="radio" status="active" checked="true"  name="egwTaskStatus" onchange="egwsetDateTimeBoxEnable()" style="vertical-align:middle;margin-right:4px;"/>
						<label for="" style="font-size:13px;"><%=rb.getString("LiJiZhiXing")%></label>
						<input type="radio" id="" status="suspend"  name="egwTaskStatus" onchange="egwsetDateTimeBoxEnable()" style="margin-left:120px;vertical-align:middle;margin-right:4px;"/>
						<label for="" style="font-size:13px;"><%=rb.getString("GuaQi")%></label>
						<input type="radio" id="egw_schedule_upgrade" status="timing" name="egwTaskStatus" onchange="egwsetDateTimeBoxEnable()" style="margin-left:120px;vertical-align:middle;margin-right:4px;"/>
						<label for="" style="margin-right:20px;font-size:13px;"><%=rb.getString("DingShiZhiXing")%></label>
						<input id="egw_schedule_timebox" class="easyui-datetimebox border-box border" style="height:26px;margin-left:30px;"
						   data-options="disabled:true,editable:false,onSelect:hiddenMes">
				   </div>
		   			<div id='egwDateMesDiv' class='egw_error_mes' style='margin-left:35px;margin-top:3px;'><%=rb.getString("QingXuanZeShiJian")%></div>
				</div>
			</div>
		</div>
	</div>
	<div class="slideFooter">
		<a class="easyui-linkbutton linkbutton linkbutton_trend" href="javascript:void(0)" onclick="saveAddegwUpgradeTask()"><span><%=rb.getString("QueDing")%></span></a>
		<a class="easyui-linkbutton linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="cancelAddegwTask()"><span><%=rb.getString("QuXiao")%></span></a>
	</div>
</div>
<%-- 升级任务工具栏 --%>
<div id="toolbar_egwAddTaskList" class="toolbarContainer">
    <div class="queryGroup" style="margin-left:15px; width: 80%;">
    	<input id="egwAddTaskSearchName" style="" placeholder="<%=rb.getString("EGWMingCheng")%>/<%=rb.getString("eGWIP")%>" style="width: calc(100% - 30px);">
		<b class='el-icon el-icon-common-search' onclick="queryegwUpgradeTaskList()"></b>
    </div>
</div>
<script>
	$(function(){
		var ele = $("#egw_schedule_timebox");
		disableSelectEarlyTime(ele);
		// 声明egw列表
		$("#egwDeviceTable").datagrid({
			border: false,
			fitColumns: true,
			fit: false,
			width: '99%',
			height:'100%',
		    rownumbers: true,
		    url: '${ctx}/egw/register/queryEgwPageList.action',
		    queryParams:{
		    	searchText:$("#egwAddTaskSearchName").val(),
		    	timeZone:timeZone
		    },
		    pageSize: 100,
		    pageList: [100],
		    striped: true,
		    singleSelect: false,
		    pagination: true,
		    pagePosition: 'bottom',
		    idField: 'gw_id',
		    checkOnSelect:true,
	        selectOnCheck:true,
		    onBeforeLoad:beforeLoad_upgrade_egwList,
		    onLoadError: datagridLoadError,
		    toolbar: '#toolbar_egwAddTaskList',
		    onLoadSuccess: loadSuccess_egw_upgrade,
		    onSelect:addSelectedegw_addTask_upgrade,
	        onUnselect:deleteSelectedegw_task_upgrade,
	        onSelectAll:addSelectedegw_addTask_upgrade_all,
	        onUnselectAll:deleteSelectedegw_task_upgrade_all,
		    columns: [[
				{field: 'ck', checkbox: true},
				{field: 'gw_id', hidden: true},
				{field: 'gw_name', sortable: true, width: 400, title: '<%=rb.getString("EGWMingCheng")%>'},
				{field: 'gw_ip', sortable: true, width: 400, title: '<%=rb.getString("eGWIP")%>'},
				{field: 'gw_port', sortable: true, width: 150, title: '<%=rb.getString("EGWDuanKou")%>'}
			]]
		});
		//已选择egw列表
		$("#selectedegwTable").datagrid({
			border: false,
	        rownumbers: true,
	        striped: true,
	        singleSelect: true,
	        fit:true,
	        fitColumns:true,
	        data:[],
	        onLoadSuccess:function(){
	        	 $("#selectedegwDiv .datagrid-view2 .datagrid-body").bind("mouseover",function(){
	        		var td = event.target
	        		if(event.target.tagName=='SPAN'||event.target.tagName=='TD'||(event.target.tagName=='DIV'&& $(event.target).hasClass('datagrid-cell'))){
	        			var td = event.target;
	        			if(event.target.tagName=='TD'){
	        				
	        			}else{
	        				td = $(event.target).parents('td')[0];
	        			}
	        			td.classList.add('visible');
	        			var grid = $("#selectedegwTable");
	        			var cwidth = grid.datagrid('getColumnOption','text').width;
	        			var pwidth = $("#selectedegwDiv .datagrid-view2").width();
	        			$($(td).find("span")).css("left",pwidth-50);
	        		}
	        	}).bind("mouseout",function(){
	        		event.target.classList.remove('visible');
	        	});
	        	$("#selectedegwDiv").bind("mouseout",function(){
	        		$('#selectedegwDiv .visible').removeClass('visible');
	        	});
	        },
	        columns: [[
				{field: 'value', hidden: true},
				{field: 'text',width:300,title:egwTitleClear,formatter:egwDeleteSelectRow}
			]]
		})
		//选择升级文件列表
	    $("#selectUpgradeFileTable").datagrid({
	    	url:'${ctx}/egw/softwareFile/querySoftwareFilePageList.action',
	    	queryParams:{timeZone:timeZone},
	    	fit:true,
	    	fitColumns:true,
	    	border:false,
	    	singleSelect:true,
	    	rownumbers:true,
	    	striped:true,
	    	pagination:true,
	    	pagePosition:'bottom',
	    	idField:'id',
	    	//onBeforeLoad:beforeLoad_fileUpgrade,
	    	//onLoadSuccess:fileinfo_sys_load_success,
	    	columns: [[
					{field: 'id',hidden:true},
					{field:"operation",width:30,fixed:true,formatter:selectegwFileFmt,align:"center"},  
	 				{field: 'version',width:270,title:'<%=rb.getString("BanBen") %>'},
	 				{field: 'product_type',width:150,title:'<%=rb.getString("ChanPinLeiXingBiaoZhi") %>'},
	 				{field: 'file_size',width:100,title:'<%=rb.getString("WenJianDaXiao") %>'},
	 				{field: 'upload_time',width:140,title:'<%=rb.getString("ShangChuanShiJian") %>'},
	    			]],
   			onSelect: function(rowIndex,row){
   				$.each($("input.egw_singleCheck"),function(index,item){
   					if(index == rowIndex) $(item).prop('checked',true);
   					else $(item).prop('checked',false);
   				})
   				$("#selectFileMesDiv").css("visibility","hidden");
   			}  
	    })
	    $("#egwAddTaskSearchName").bind("keyup",function(e){
	    	if(e.keyCode == 13){
	    		queryegwUpgradeTaskList();
	    	}
	    })
	})
	// 执行方式选择
	function egwsetDateTimeBoxEnable(){
		if (document.getElementById("egw_schedule_upgrade").checked) {
			$("#egw_schedule_timebox").datetimebox("enable");
		} else {
			$("#egw_schedule_timebox").datetimebox("disable");
			$("#egwDateMesDiv").css("visibility","hidden");
		}
	}
	// 已选列表 表头格式化
	function egwTitleClear(){
		var span = $('<p onclick="egwClearAllSelect()" style="position:absolute;top:0px;right:0px;line-height:38px;cursor:pointer;"><span style="float:right;color:#4D84FF;margin-right:10px;"><%=rb.getString("QingKong")%></span><span style="font-size:16px;margin-top:10px;margin-right:5px;" class="el-icon el-icon-operation-delete"></span></p>');
		var title = $(this);
		title.after(span);
		return "<%=rb.getString("EGWMingCheng")%><%=rb.getString("ZuoKuoHao")%><%=rb.getString("eGWIP")%><%=rb.getString("YouKuoHao")%>"
	}
	// 清空
	function egwClearAllSelect(){
		$("#selectedegwTable").datagrid("loadData", {total:0,rows:[]});
		$("#egwDeviceTable").datagrid("clearSelections");
	}
	/**
	* 网关名称 数据格式化
	* @param value{string} 绑定值
	* @param rowData{object}  行数据
	* @param rowIndex{number}  下标
	*/
	function egwDeleteSelectRow(value,rowData,rowIndex){
		var delStr = "<span onclick='deleteegwRow(\""+rowData.value+"\")' class='upgrade_delete'></span>"
		return value+delStr;
	}
	// 移除已选
	function deleteegwRow(value){
		var grid = $.data($("#egwDeviceTable")[0],'datagrid');
		grid.selectedRows = grid.selectedRows.filter(function(item){
			return item["gw_id"]!=value;
		})
		grid.checkedRows = grid.checkedRows.filter(function(item){
			return item["gw_id"]!=value;
		})
		var rows = $("#selectedegwTable").datagrid("getRows");
		rows.map(function(item,index){
			if(value == item.value){
				$("#selectedegwTable").datagrid("deleteRow",index);
			}
		})
		var allRows = $("#egwDeviceTable").datagrid("getRows");
		allRows.map(function(item,index){
			if(value == item["gw_id"]){
				 $("#egwDeviceTable").datagrid("unselectRow",index);
			}
		})
	}
	// 勾选按钮格式化
	function selectegwFileFmt(){
		var html = '<input type=\"radio\" class="egw_singleCheck"/>';
		return html;
	}
	// 数据请求前 赋值操作
	function beforeLoad_upgrade_egwList(param){
		param["searchText"] = $("#egwAddTaskSearchName").val();
		$("#egwDeviceTable").datagrid("getPager").pagination({
			layout:['prev','manual','next','refresh']
		});
	}
	// 数据加载成功
	function loadSuccess_egw_upgrade(){
		$(this).datagrid("fixRownumber");
		$(this).datagrid("enableContextmenuAutoSize");
	}
	/**
	* 数据勾选
	* @param index{number} 下标
	* @param row{object}  行数据
	*/ 
	function addSelectedegw_addTask_upgrade(index,row){
		var grid = $("#selectedegwTable");
		var cwidth = grid.datagrid('getColumnOption','text').width;
		var pwidth = $("#selectedegwDiv .datagrid-view2").width();
		$("#deviceMesDiv").css("visibility","hidden");
		var row = {
				value:row["gw_id"],
				text:row["gw_name"]+"("+row["gw_ip"]+")"
		}
		grid.datagrid("appendRow", row);
		var cData = grid.datagrid("getRows");
		if(pwidth - cwidth >10){
			var cl = $("#selectedegwTable").datagrid('getColumnOption','text');
			cl.width = pwidth;
			$("#selectedegwTable").datagrid({fitColumns:true,data:cData});
		}else{
			$("#selectedegwTable").datagrid({fitColumns:false,data:cData});
		}
	}
	// 全选
	function addSelectedegw_addTask_upgrade_all(){
		var selCell = $("#egwDeviceTable").datagrid("getSelections");
		var del = $("#selectedegwTable").datagrid("getRows");
		var selArr = [];
		del.map(function(item,index){
			selArr.push(item.value);
		})
		selCell.map(function(item,index){
			if(selArr.indexOf(item["gw_id"])!=-1){
				return;
			}else{
				var row = {
						value:item["gw_id"],
						text:item["gw_name"]+"("+item["gw_ip"]+")"
				}
				$("#selectedegwTable").datagrid("appendRow", row);
			}
		})
	}
	/**
	* 取消数据勾选
	* @param index{number} 下标
	* @param row{object}  行数据
	*/ 
	function deleteSelectedegw_task_upgrade(index,row){
		var gwId = row["gw_id"];
		var allRows = $("#selectedegwTable").datagrid("getRows");
		allRows.map(function(item,index){
			if(item.value ==gwId ){
				$("#selectedegwTable").datagrid("deleteRow",index);
			}
		})
	}
	// 取消全选
	function deleteSelectedegw_task_upgrade_all(){
		$("#selectedegwTable").datagrid("loadData", {total:0,rows:[]});
	}
	// 任务名称 验证
	function egw_checkUpgradeTaskName(){
		var value = $('#egwUpgradeTaskName').val();
		if(value == ""){
			$('#egwUpgradeTaskName').addClass('egwErrorborder');
			$($('#egwUpgradeTaskName').siblings("p")).css("visibility","visible");
		}else{
			$('#egwUpgradeTaskName').removeClass('egwErrorborder');
			$($('#egwUpgradeTaskName').siblings("p")).css("visibility","hidden");
		}
	}
	// 新建保存
	function saveAddegwUpgradeTask(){
		egw_checkUpgradeTaskName();
		var selegws = $("#selectedegwTable").datagrid("getRows");
		if (selegws.length == 0) {
			$("#deviceMesDiv").css("visibility","visible");
		}else{
			$("#deviceMesDiv").css("visibility","hidden");
		}
		var selectFile = $("#selectUpgradeFileTable").datagrid("getSelections");
		if(selectFile.length == 0){
			$("#selectFileMesDiv").css("visibility","visible");
		}else{
			$("#selectFileMesDiv").css("visibility","hidden");
		}
		var timebox = $("#egw_schedule_timebox").datetimebox("getValue");
		var isCheck = $("#egw_schedule_upgrade")[0].checked;
		if(isCheck&&timebox==""){
			$("#egwDateMesDiv").css("visibility","visible");
		}else{
			$("#egwDateMesDiv").css("visibility","hidden");
		}
		var isPassFlag = true;
		$('.egw_error_mes').each(function(index,item){
			if($(item).css("visibility") != "hidden"){
				isPassFlag = false;
			}
		})
		if(isPassFlag){
			var param = {};
			param["timeZone"] = timeZone;
			param["taskName"] = $("#egwUpgradeTaskName").val();
			var egwStr = "";
			var selectEgw = $("#selectedegwTable").datagrid("getRows");
			selectEgw.map(function(item,index){
				egwStr += item.value + ",";
			})
			param["selectEgwList"] = egwStr;
			var row = $("#selectUpgradeFileTable").datagrid("getSelected");
			param["softwareId"] = row.id;
			var status = document.getElementsByName("egwTaskStatus");
			for (var i = 0; i < status.length; i++) {
				if (status[i].checked == true) {
					param["exeMode"] = $(status[i]).attr("status");
				}
			}
			if(param["exeMode"] == "timing"){
				param["exeTime"] = $("#egw_schedule_timebox").datetimebox("getValue");
			}else{
				param["exeTime"] = "";
			}
			$.post("${ctx}/egw/softwareUpgrade/addSoftwareInfos.action",param,function(data){
				if(data["success"]){
					showMsg('success_msg','<%=rb.getString("ChengGong")%>');
					cancelAddegwTask();
					//$("#egwUpgradeListTable").datagrid("reload");
					toViewEGW();//调整到任务查看页面 
					
				}else{
					showMsg('error_msg',data["message"]);
				}
			},"json")
		}
	}
	function hiddenMes(date){
		$("#egwDateMesDiv").css("visibility","hidden");
	}
	// 搜索
	function queryegwUpgradeTaskList(){
		$("#egwDeviceTable").datagrid("reload");
	}
</script>