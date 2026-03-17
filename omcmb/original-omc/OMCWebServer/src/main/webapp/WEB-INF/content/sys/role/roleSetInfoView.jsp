<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	#sysViewRoleFeatureLimit .datagrid-row-over > td {
  		background: #fff !important;
  		cursor:pointer;
	}
	#sysViewSourceDiv .tree-node {
	  background: #fff !important;
	  cursor:pointer;
	  padding:3px 0px 3x 0px;
	}
</style>
<div class='el-card__header'>
	<span><%=rb.getString("JueSe")%>&nbsp;&nbsp;<span id="sysViewRoleHeader"></span></span>
	<span class='el-icon el-icon-close' onclick='closeViewSysUserGrooupWindow()'></span>
</div>
<div class='el-card__body' style='display:flex;flex-direction:column;flex:1 1 auto;height:100%;overflow:auto;'>
	<div style='background:#fff;border:1px solid #EEE'>
		<div class="omcPageTitleDiv_contain" style='margin-top:20px;'>
			<ul class="omcPageTitleContainer_contain" style="width:80%">
				<li class="active"><%=rb.getString("GongNengQuanXianLieBiao")%></li>
			</ul>
		</div>
		<div class="deviceLogContainer" style='margin-top:15px;'>
			<div id='sysViewRoleFeatureLimit' class="tree-lines" style='user-select:none;height:450px;width:787px;border:1px solid #CCE1EF;margin-top:10px;overflow:auto'>
				<table style='height:100%;' id='sysViewFeaureLimitTable'></table>
			</div>
		</div>
		<div class="omcPageTitleDiv_contain" style='margin-top:20px;'>
			<ul class="omcPageTitleContainer_contain" style="width:80%">
				<li class="active"><%=rb.getString("ZiYuanQuanXianLieBiao")%></li>
			</ul>
		</div>
		<div class="deviceLogContainer" style='margin-top:15px;'>
			<div id='sysViewSourceDiv' class="tree-lines" style='padding-left:10px;user-select:none;height:450px;width:777px;border:1px solid #CCE1EF;margin-top:10px;overflow:auto'>
				<ul id='sysViewSourceUl'></ul>
			</div>
		</div>
	</div>
</div>
<script>
	var check = $("#sysRoleSetTable").datagrid("getSelections")[0];
	if($("#curr_operator_span").text() == ""){
		var operator_code = "default";
	}else{
		var operator_code = operator_code;
	}
	$(function(){
		closeLoading();
		$("#sysViewRoleHeader").html(check.role_name);
 		$('#sysViewFeaureLimitTable').treegrid({
	        url:"${ctx}/sys/role/getFeature.action",
	        queryParams:{
	        	"role_id":check.role_id,
				"type":"view",
				"operator_code":operator_code
	        },
	        idField: 'id',            //定义关键字段来标识树节点。也就是数据的id
	        treeField: 'text',        //定义树形显示字段
	        fitColumns:true,
	        animate:true,
	        columns: [[                //定义表格头名称
	            {
	                title: '<%=rb.getString("QuanXianLieBiao")%>',
	                field: 'text',
	                width:50
	            },
	            {
	                title: '<%=rb.getString("GongNengCaoZuo")%>',
	                field: 'operation',
	                width: 50,
	                formatter: sysViewOnlyReadAndWriteFormat
	            }
	        ]],
	        onLoadSuccess:function(node,data){
	        	$("#sysViewRoleFeatureLimit table .datagrid-btable:first>tbody>tr>td>div>.tree-file").removeClass("tree-file").addClass("tree-folder");
	        	var treetable = $(this).treegrid('getPanel').find('div.datagrid-body').parent().parent().parent();
	        	treetable.css('border-width','0px');
	        	var tr = $(this).treegrid('getPanel').find('div.datagrid-body tr.datagrid-row');
	        	 tr.each(function(){
	        		var td = $(this).children('td');
	        		td.css({
	        			'border-width':'0px'
	        		})
	        	});
	        },
	        onBeforeSelect:function(){
	        	return false;
	        },
	        onSelect:function(){
	        	return false;
	        },
	        loadFilter: proccessData,
	        onClickRow:function(row){
	        	$('#sysViewFeaureLimitTable').treegrid("toggle", row.id);
	        } ,
	        onExpand:function(row){
	      		var root = $('#sysViewFeaureLimitTable').treegrid('getRoot');
	      		if(row.id!=root.id){
	      			var children = $('#sysViewFeaureLimitTable').treegrid('getRoot').children;
	      			children.map(function(item,index){
	      				if(item.id == row.id){
	      					$('#sysViewFeaureLimitTable').treegrid("expand", item.id);
	      				}else{
	      					$('#sysViewFeaureLimitTable').treegrid("collapse", item.id);
	      				}
	      			})
	      		}
	      	}
	    });
 	  	  $('#sysViewSourceUl').tree({
 				url:"${ctx}/sys/role/getDeviceGroup.action",
 				queryParams:{
 					"type":"view",
 					"role_id":check.role_id,
 					"operator_code":operator_code
 		        },
 				checkbox:false,
 				idField: 'id',            //定义关键字段来标识树节点。也就是数据的id
 				treeField: 'text',        //定义树形显示字段
 				fitColumns:true,
 				animate:true,
 				loadFilter: proccessData,
 				onLoadSuccess:function(node,data){
 					var root = $('#sysViewSourceUl').tree("getRoots");
 					var children = $('#sysViewSourceUl').tree("getChildren",root);
 					children.map(function(item,index){
 						var isLeaf = $('#sysViewSourceUl').tree("isLeaf",item.target);
 						if(!isLeaf && index == 1 || index == 0){
 							$('#sysViewSourceUl').tree("expand",item.target);
 						}else{
 							$('#sysViewSourceUl').tree("collapse",item.target);
 						}
 					})
 				} ,
 				 onBeforeSelect:function(){
 			        	return false;
 			        },
 			     onSelect:function(){
 			        	return false;
 			        },
 				onClick:function(node){
 					$('#sysViewSourceUl').tree("toggle", node.target);
 				},
 				onCheck:function(){
 					$("#sysViewSourceUl").css("visibility","hidden");
 				},
 				onExpand:function(node){
 					var nodeAll = $('#sysViewSourceUl').tree("getRoots")[0].children;
 					nodeAll.map(function(item,index){
 						if(node.id == item.id){
 								$('#sysViewSourceUl').tree("expand",node.target);
 						}else{
 							$('#sysViewSourceUl').tree("collapse",item.target);
 						}
 					})
 				}
 				});
	});
	function sysViewOnlyReadAndWriteFormat(value,row,rowIndex){
		var rowId = row.id;
		if(row.children && row.children.length>0){
			return "";
		}else{
			if(row.text == "EPC"||row.text == "告警确认"||row.text == "Alarm Confirm"||row.text == "告警清除"||row.text == "Clear Alarm"||row.text == "告警恢复"||row.text == "Restore Alarm"||row.text == "告警删除"||row.text == "Delete Alarm"||row.text == "用户向导"||row.text == "User Guide"||row.text == "关于"||row.text == "About"||row.text == "Dashboard"||row.text == "首页"
				||row.text == "同步"||row.text == "Synchronize"||row.text == "关联的CPEs"||row.text == "CPEs"
					||row.text == "基站IMSI信息清除"||row.text == "Clear IMSI"||row.text == "系统资源监控"||row.text == "Resource"||row.text == "设置"||row.text == "Setting"||row.text == "激活"||row.text == "Active"||row.text == "eGW"||row.text == "网关"||row.text == "UPS"||row.text == "APN"||row.text == "快速配置"||row.text == "Quick Configuration"){
				return "";
			}
			if(row.reWrite == "true"){
				if(row.write){
					return "<span style='cursor:pointer;margin-left:10px;' class='tree-checkbox tree-checkbox3 readOper '></span><span class='tree-title'><%=rb.getString("ZhiDu")%></span>&nbsp;&nbsp;<span style='cursor:pointer;margin-left:5px;' class='tree-checkbox tree-checkbox3 writeOper'></span><span class='tree-title'><%=rb.getString("ZhiXie")%></span>"
				}else{
					return "<span style='cursor:pointer;margin-left:10px;' class='tree-checkbox tree-checkbox3 readOper '></span><span class='tree-title'><%=rb.getString("ZhiDu")%></span>&nbsp;&nbsp;<span style='cursor:pointer;margin-left:5px;' class='tree-checkbox tree-checkbox4 writeOper'></span><span class='tree-title'><%=rb.getString("ZhiXie")%></span>"
				}
			}else{
				return "<span style='cursor:pointer;margin-left:10px;' class='tree-checkbox tree-checkbox3 readOper '></span><span class='tree-title'><%=rb.getString("ZhiDu")%></span>";
			}
			
		}
	}
</script>
