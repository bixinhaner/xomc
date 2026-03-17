<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
	#viewFeatureLimitDiv .datagrid-row-over > td {
  		background: #fff !important;
  		cursor:pointer;
	}
	.headContent{
		font-size:14px;
		color:#363B4E;
		font-weight:700;
		margin-top:30px;
		display:flex;
		align-items:center;
	}
	.headContent img{
		margin-right:14px;
	}
</style>
<div class="slidebarTitleDiv">
	<span class="slideTitle"><%=rb.getString("JueSe")%></span>&nbsp;&nbsp;<span id='operRoleViewHead'></span>
	<div class="slideIcon el-icon el-icon-close" onclick='closeViewRoleWindow()'></div>
</div>
<div class="slideBody">
	<div class="slideCont">
		<div class="omcPageTitleDiv_contain">
			<div class="headContent">
				<img src="${ctx}/css/images/global/settingBetter.png"><%=rb.getString("GongNengQuanXianLieBiao") %>
			</div>
		</div>
		<div class="deviceLogContainer" style='margin-top:20px;'>
			<div id="op_view_feature" style="width: 100%;">
				<el-ctree ref='featureViewTree' style="height: 500px;"
					:readonly="true"
					title="<%=rb.getString("QuanXianLieBiao")%>"
					:default-expanded-keys="['0']"
					:data="data1"
					:check-forms="[
						{key: 'forms',label: '<%=rb.getString("ZhiDuQuanXuan")%>',prop: 'checked'},
						{key: 'wForms',label: '<%=rb.getString("KeXieQuanXuan")%>',prop: 'write'}
					]"
					:ignore="{
						forms: ['1','7','10','30','32','35','38','64','66','76','81','92','96','97','98'],
						wForms: ['1','58','59','60','62','63','94','103']
					}"></el-ctree>
			</div>
		</div>
		<div class="onlyLocalShow">
			<div class="omcPageTitleDiv_contain" style='margin-top:20px;'>
				<div class="headContent">
					<img src="${ctx}/css/images/global/settingBetter.png"><%=rb.getString("YunYingShangLieBiao") %>
				</div>
			</div>
			<div class="deviceLogContainer">
				<div style='height:424px;border:1px solid #E9E9E9;margin-top:20px;'>
					<table  id='operViewRoleOperator'></table>
				</div>
			</div>
		</div>
	<div class="omcPageTitleDiv_contain">
</div>
<script>
	var roleId = $('#operRoleSetListTable').datagrid("getSelections")[0].role_id;
	var roleName = $('#operRoleSetListTable').datagrid("getSelections")[0].role_name;
	var opvm = new Vue({
		el: '#op_view_feature',
		data() {
			return {
				data1: []
			}
		},
		methods: {
			init(data) {
				var vm = this;
				
				vm.initReadStatus(data);
				
				vm.data1 = data;
	        	vm.$nextTick(function(){
	        		vm.$refs.featureViewTree.reviewForms(vm.data1);
	        	});
			},
			initReadStatus(data) {
				var vm = this;
				data.map(function(row){
					if(row.children && row.children.length) {
						vm.initReadStatus(row.children);
					}
					row.checked = row.write === true || row.write === false;
				})
			},
			getResult() {
				var vm = this;
					res = vm.$refs.featureTree.getResult(),
					menu_ids = [];
				
				res.wForms.map(function(id){
					id!='0' && menu_ids.push({id: id,write: true});
				});
				res.forms.map(function(id){
					if(!res.wForms.includes(id) && id!='0'){
						menu_ids.push({id: id,write: false})
					}
				})
				
				return menu_ids;
			}
		},
		mounted() {
			var vm = this;
			axios.post("${ctx}/system/superRoleSet/getFeature.action",stringify({
	        	"role_id":roleId,
				"type":"view"
	        })).then(function(data){
				vm.init(data.data);	
			})
		}
	});
	$(function(){
		closeLoading();
		
		if(isCloudCore == "true" && roleId == "2"){
			$(".onlyLocalShow").hide();
		}else{
			$(".onlyLocalShow").show();
		}
		
		$("#operRoleViewHead").html(roleName);
 		$('#operViewFeatureLimitTable').treegrid({
	        url:"${ctx}/system/superRoleSet/getFeature.action",
	        queryParams:{
	        	"role_id":roleId,
				"type":"view"
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
	                formatter: operViewOnlyReadAndWriteFormat
	            }
	        ]],
	        onLoadSuccess:function(node,data){
	        	$("#viewFeatureLimitDiv table .datagrid-btable:first>tbody>tr>td>div>.tree-file").removeClass("tree-file").addClass("tree-folder");
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
	        	$('#operViewFeatureLimitTable').treegrid("toggle", row.id);
	        } ,
	        onExpand:function(row){
	      		var root = $('#operViewFeatureLimitTable').treegrid('getRoot');
	      		if(row.id!=root.id){
	      			var children = $('#operViewFeatureLimitTable').treegrid('getRoot').children;
	      			children.map(function(item,index){
	      				if(item.id == row.id){
	      					$('#operViewFeatureLimitTable').treegrid("expand", item.id);
	      				}else{
	      					$('#operViewFeatureLimitTable').treegrid("collapse", item.id);
	      				}
	      			})
	      		}
	      	}
	    });
 		$("#operViewRoleOperator").datagrid({
			url:'${ctx}/system/superRoleSet/getOperatorListByRole.action',
			queryParams : {
				"role_id":roleId,
				"type":"view"
				},
			singleSelect:true,
			fit:true,
			fitColumns:true,
			border:false,
			rownumbers:true,
			pagePosition:'bottom',
			pagination: true,
			striped: true,
			onLoadSuccess:datagridLoadSuccess,
			columns: [[
				{field: 'operator_code',hidden:"true"},
				{field: 'operator_name',width:100,title:'<%=rb.getString("YunYingShangMingCheng")%>'},
				{field: 'cloud_key',width:100,title:'CloudKey'},
			]]
		});
	});
	function proccessData(data){
		data.map(function(item){
    		if(item.children && item.children.length>0) {
    			proccessData(item.children);
    		}else{
    			delete item.state;
    		}
    	});
    	return data;
	}
	function operViewOnlyReadAndWriteFormat(value,row,rowIndex){
		var rowId = row.id;
		if(row.children && row.children.length>0){
			return "";
		}else{
			if(row.text == "EPC"||row.text == "告警确认"||row.text == "Alarm Confirm"||row.text == "告警清除"||row.text == "Clear Alarm"||row.text == "告警恢复"||row.text == "Restore Alarm"||row.text == "告警删除"||row.text == "Delete Alarm"||row.text == "用户向导"||row.text == "User Guide"||row.text == "关于"||row.text == "About"||row.text == "首页"||row.text == "Dashboard"
				||row.text == "同步"||row.text == "Synchronize"||row.text == "关联的CPEs"||row.text == "CPEs"
					||row.text == "基站IMSI信息清除"||row.text == "Clear IMSI"||row.text == "系统资源监控"||row.text == "Resource"||row.text == "设置"||row.text == "Setting"||row.text == "Setting"||row.text == "激活"||row.text == "Active"||row.text == "eGW"||row.text == "网关"||row.text == "APN"||row.text == "快速配置"||row.text == "Quick Configuration"){
				return "";
			}
			if(row.write){
				return "<span style='cursor:pointer;margin-left:10px;' class='tree-checkbox tree-checkbox3 readOper '></span><span class='tree-title'><%=rb.getString("ZhiDu")%></span>&nbsp;&nbsp;<span style='cursor:pointer;margin-left:5px;' class='tree-checkbox tree-checkbox3 writeOper'></span><span class='tree-title'><%=rb.getString("ZhiXie")%></span>"
			}else{
				return "<span style='cursor:pointer;margin-left:10px;' class='tree-checkbox tree-checkbox3 readOper '></span><span class='tree-title'><%=rb.getString("ZhiDu")%></span>&nbsp;&nbsp;<span style='cursor:pointer;margin-left:5px;' class='tree-checkbox tree-checkbox4 writeOper'></span><span class='tree-title'><%=rb.getString("ZhiXie")%></span>"
			}
		}
	}
</script>