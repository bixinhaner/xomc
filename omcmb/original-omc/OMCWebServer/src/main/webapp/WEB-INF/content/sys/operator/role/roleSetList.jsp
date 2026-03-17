<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<style type="text/css">
.textbox.combo .textbox-text {
	padding: 0 4px !important;
}
#role_set_form .textbox.combo .textbox-text {
	padding: 4px !important;
}
.omcLogGroup2 .textbox.combo .textbox-text {
	padding: 0 4px !important;
}
.textbox.combo{
	vertical-align:top;
}
.deviceLogContainer{
	width:90%;
	margin-left:60px;
	margin-top:20px;
}
.deviceListContainer,.selectedDevicesList{
	width:45%;
	height:100%;
	
}
.flex-ctn{
	display: -webkit-flex;
	display: flex;
	flex-direction: column;
}
.flex-item {
	flex: auto;
	overflow: auto;
}
.titleButtonText {
	transition: opacity 0.5s ease-in;
}
.inputformat label{
	display:block;
	font-size:12px;
	color:#85A8BF;
}
.inputformat input{
	width:400px;
	height:25px;
	border:1px solid #85A8BF;
	display:block;
	margin-top:6px;
	margin-bottom:6px;
}
.inputformat span{
	color:#CC0000;
}
.tree-title{
	cursor:pointer;
}
.tree-icon{
	cursor:pointer;
}
.tree-checkbox:{
	cursor:pointer;
}
</style>
<div class="panelDefault" style="overflow:hidden">
	<!-- 关闭按钮 -->
    <div id='operCloseCircle' class="addConfig circleIcon" style="display:none"> 
    	<span class="circleBg close_circle"  onclick="cancelOperModifyRoleDiv()"></span>
    	<div class="titleButtonText"><%=rb.getString("GuanBi")%></div>
    </div>
    <div class="tabsTitle">
    	<span class='active'><%=rb.getString("RoleSet")%></span>
	</div>
    <div class="singleContentDiv">
		<table id="operRoleSetListTable"></table>
	</div>
	
	<!--修改角色  -->
    <div id="operModifyRoleDiv" class='slidebarPanel slide-position-top'></div>
    
    <!-- 查看角色 -->
    <div id='operViewRoleDiv' class='slidebarPanel slide-position-top'></div>
</div>

<!-- 工具栏-  -->
 <div id="toolbar_operSearchRoleList" class="toolbarContainer" style="position:relative">
      <div class="queryGroup">
      	<input placeholder="<%=rb.getString("JueSeMingCheng")%>" id='operRoleName'>
		<b class="el-icon el-icon-common-search" onclick="$('#operRoleSetListTable').datagrid('reload')"></b>
      </div>
</div> 

<script type="text/javascript">
if($("#curr_operator_span").text() == ""){
	var operator_code = "default";
}else{
	var operator_code = operator_code;
}

$(function() {
    closeLoading();
    $("#operRoleSetListTable").datagrid({
		url:'${ctx}/system/superRoleSet/getSuperRoleSet.action',
		queryParams : {
			timeZone:timeZone,
			'operator_code':operator_code,
			'role_name':$('#operRoleName').val()
			},
		singleSelect:true,
		fit:true,
		fitColumns:true,
		border:false,
		rownumbers:true,
		pagePosition:'bottom',
		pagination: true,
		striped: true,
		toolbar:"#toolbar_operSearchRoleList",
		onLoadSuccess:datagridLoadSuccess,
		onBeforeLoad:beforeLoad_roleSetList,
		columns: [[
			{field: 'role_name',width:100,title:'<%=rb.getString("JueSeMingCheng")%>'},
			{field: 'upd_user',width:100,title:'<%=rb.getString("QuanXianXiuGaiRen")%>'},
			{field: 'upd_time',width:100,title:'<%=rb.getString("XiuGaiShiJian")%>'},
			{field: 'role_desc',width:100,title:'<%=rb.getString("MiaoShu")%>'},
			{field: 'operation',width:100,title:'<%=rb.getString("CaoZuo")%>',formatter:roleOperFormatter}
		]]
	});
    
    $("#operRoleName").bind("keyup", function(e){
  		if (e.keyCode == 13){
  			$('#operRoleSetListTable').datagrid('reload');
  		}
  	}); 
});

function roleOperFormatter(value, rowData, rowIndex){
	value = "<div class='el-icon el-icon-operation-view' title='<%=rb.getString("ChaKan") %>' style='display:inline-block;margin-left:5px;cursor:pointer;' onclick ='openOperViewRoleDiv()' ></div>";
	if("${issubadmin}" == 1){
	}else{
		value += "<div class='el-icon el-icon-operation-edit CODE_OPERATOR_ROLE hidden' title='<%=rb.getString("XiuGai") %>' style='margin-left:20px;display:inline-block;cursor:pointer;' onclick='openOperModifyRoleDiv()'></div>";
	}
	return value;
}

//修改角色
function openOperModifyRoleDiv(){
	$('#operModifyRoleDiv').panel({
		href:"${ctx}/system/superRoleSet/toModify.action",
		width:878
	})
	$('#operModifyRoleDiv').animate({right:"0px"},450);
}

function cancelOperModifyRoleDiv(){
	$('#operModifyRoleDiv').animate({right:"-900px"},300);
}

function showTipCircle(ele){
		$(ele).next().css('opacity','1');
}

function hideTipCircle(ele){
		$(ele).next().css('opacity','0');
}
function checkboxToggle(e,ele,rowId){
	e.stopPropagation();
	var treeData = $('#operModifyFeatureLimitTable').treegrid("getData");
	var leafLength = $('#operModifyFeatureLimitTable').treegrid("getChildren",rowId).length;
	//未勾选状态
	if($(ele).hasClass('tree-checkbox0') || $(ele).hasClass('tree-checkbox2')){
		$(ele).removeClass('tree-checkbox0').addClass('tree-checkbox1');
		//$($(ele).closest("td").prev()[0]).find(".treeOper").removeClass("tree-checkbox1").addClass("tree-checkbox0").trigger("click");
	}else{
		//取消勾选状态
		$(ele).removeClass('tree-checkbox1').addClass('tree-checkbox0');
	}
	updateWritableStatus(rowId);
}
function updateWritableStatus(nodeId) {
	var $el = $('[nodeid='+nodeId+']'),
		pid = $el.attr('pid'),
		node = $('[writeid='+nodeId+']')[0],
		pckbox = $('[writeid='+pid+']')[0],
		childs = $('[writepid='+pid+']'),
		checkedClass = 'tree-checkbox1',
	    middleClass = 'tree-checkbox2',
	    unCheckClass = 'tree-checkbox0',
	    excludes = 'Dashboard,首页',
	    status = Array.from(node.classList).includes(checkedClass);

	downAccess(nodeId,status,excludes);
	upAccess(pid);
	cascade(nodeId);
	
	// 级联批量只读
	function cascade(id){
		var nodeEl = $('[nodeid='+id+']')[0],
			readEl = $('[readid='+id+']')[0],
			writeEl = $('[writeid='+id+']')[0],
			nodeChecked = Array.from(nodeEl.classList).includes(checkedClass);
		
		if(readEl){ // 是对批量操作的行
			var clsList = Array.from(readEl.classList),
				readChecked = clsList.includes(checkedClass),
				isReadMiddle = clsList.includes(middleClass);
			if(status) {
				if(!readChecked) {// 批量可写为选中，且批量只读为非选中
					readEl.click();
				}
				// 更新目录状态
				accessClass(nodeEl,checkedClass,excludes);
			}else {
				// 更新目录状态
				if(isReadMiddle || readChecked) {
					accessClass(nodeEl,middleClass,excludes);
				}else {
					accessClass(nodeEl,unCheckClass,excludes);
				}
			}
		}else{ // 对叶子节点的操作
			if(status) { // 叶子节点可写勾选时
				if(!nodeChecked) {
					nodeEl.click();
				}
			}else{// 取消叶子节点可写时
				var pTr = $(node).parents('tr:first'),
	            	readObj = pTr.find('.readOper')[0],
	            	readStatus = Array.from(readObj.classList).includes(checkedClass);
			
				if(readStatus) accessClass(nodeEl,middleClass,excludes);
			}
		}
	}
	// 向上递归
	function upAccess(pId){
		if(!pId) return;
		var isAll = true, has = false,
		    pckbox = $('[writeid='+pId+']:visible').get(0);
		if(pckbox) { // 遍历子节点，子节点的pid属性指向父级id
			var dirnode = $('[nodeid='+pId+']').get(0),
	    		rckbox = $('[readid='+pId+']').get(0);
		    $('[writepid='+pId+']').each(function(n,item){
		      var clist = Array.from(item.classList),
		          status = clist.includes(checkedClass);
		      
		      if(!status) isAll = false;
		      else has = true;
		      if(clist.includes(middleClass)) has = true;
		    });
		
		    if(isAll) {
		    	accessClass(pckbox,checkedClass,excludes);
		    	accessClass(rckbox,checkedClass,excludes);
		    }
		    else if(has) accessClass(pckbox,middleClass,excludes);
		    else accessClass(pckbox,unCheckClass,excludes);
		    
		    // 更新父级目录状态
		    var wStatus = Array.from(pckbox.classList).includes(checkedClass),
		    	rStatus = Array.from(rckbox.classList).includes(checkedClass),
		    	wmStatus = Array.from(pckbox.classList).includes(middleClass),
		    	rmStatus = Array.from(rckbox.classList).includes(middleClass);
		    if(wStatus && rStatus) {  // 批量只读和只写全为勾选时
		    	accessClass(dirnode,checkedClass,excludes);
		    }else if(wStatus || rStatus || wmStatus || rmStatus){
		    	accessClass(dirnode,middleClass,excludes);
		    }else { // 批量只读和只写全为不勾选时
		    	accessClass(dirnode,unCheckClass,excludes);
		    }
		    
		    // 递归处理父级节点状态: 当前节点可能是叶子或批量只读
		    var cpid = $(pckbox).attr('writepid');
		    upAccess(cpid);
		}
	}
	// 向下递归
	function downAccess(id,status,excludes){
		$('[writepid='+id+']').each(function(n,item){
		  //if(status) {accessClass(item,checkedClass,excludes);
		  //else accessClass(item,unCheckClass,excludes);
		  
		  var itemChecked = Array.from(item.classList).includes(checkedClass),
		  	  isVisible = getComputedStyle(item).display != 'none';
		  if(status != itemChecked) {
			  if(isVisible) item.click();
			  else if(!itemChecked){
				  item.click();
			  }
		  }
		  
		  downAccess($(item).attr('writeid'),status,excludes);
		});
	}
	/**
	* 设置状态的内部方法
	**/
	function accessClass(node,cls,excludes){
    	if(excludes) excludes = excludes.split(',');
    	else excludes = [];

        if(node) {
          var classes = [checkedClass,middleClass,unCheckClass];
          classes.map(function(item){
            if(item == cls) node.classList.add(item);
            else node.classList.remove(item);
          });
        }
    }
}
function openOperViewRoleDiv(){
	$('#operViewRoleDiv').panel({
		href:"${ctx}/system/superRoleSet/toView.action",
		width:878
	})
	$('#operViewRoleDiv').animate({right:"0px"},450);
}

function closeViewRoleWindow(){
	$('#operViewRoleDiv').animate({right:"-900px"},300);
}

function beforeLoad_roleSetList(param){
	param["role_name"] = $('#operRoleName').val();
}
</script>
