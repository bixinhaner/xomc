<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<style>
	.el-input__icon{
		line-height:28px;
	}
	.el-form-item{
		margin-bottom:10px;
	}
	.el-input.is-disabled .el-input__inner{
		background-color:#fff;
		border:1px solid #c9d1d6;
		color:#000;
	}
	.el-select:hover .el-input__inner,.el-select .el-input__inner{
		border:1px solid #c9d1d6;
	}
	.el-card__body{
		background:#fff;
		padding:10px 0px 0px 20px;
	}
	.el-card__footer{
		border-top:none;
		padding-left:0px;
		padding-bottom:20px;
	}
	.el-card__header{
		padding:0px;
		margin:0 20px;
	}
</style>
<div id='importConfigContent' style="height: 100%;">
	<div>
		<el-form :model='ruleForm' :rules='rules' ref='ruleForm' label-position='top'>
			<el-form-item label='<%=rb.getString("DaoRuLeiXing")%>' prop='type'>
				<el-select v-model='ruleForm.type' style='width:350px;'>
					<el-option v-for='item in options' :key='item.value' :label='item.label' :value='item.value'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label='<%=rb.getString("DaoRuWenJian") %>' prop='file'>
				<el-input v-model='ruleForm.file' :disabled="true" style='width:350px;'>
					<i @click='importFile' slot='suffix' style='display:inline-block;width:28px;height:28px;margin:4px -9px 0 0;' class='el-icon el-icon-operation-import'></i>
				</el-input>
			</el-form-item>
		</el-form>
	</div>
	<div size="mini" style="margin-top: 70px;">
		<span style="margin-right: 10px;">
			<el-dropdown @command="exportTemp">
				<el-button>
					模板导出<i class="el-icon-arrow-down el-icon--right"></i>
				</el-button>
				<el-dropdown-menu slot="dropdown">
					<el-dropdown-item command="ALL"><%=rb.getString("TongYong") %></el-dropdown-item>
					<el-dropdown-item command="QAFB">QAFB</el-dropdown-item>
				</el-dropdown-menu>
			</el-dropdown>
		</span>
		<el-button type="primary" @click="submit"><%=rb.getString("QueDing") %></el-button>
		<el-button @click="close"><%=rb.getString("QuXiao") %></el-button>
	</div>
</div>
<%-- 上传文件的用的表单 --%>
<form enctype="multipart/form-data" method="post" id="strategyConfigPlanForm">
    <input name="fileSize"  value="" hidden="true">
    <input name="operType" value="" hidden="true">
    <input name="uploadFile"  id="strategy_configPlanFile"  type="file" style="display: none;">
</form>
<%-- 下载模板用的表单 --%>
<form id="exportBatchConfigForm" style="display:none" method="post"></form>
<script>
new Vue({
	el:'#importConfigContent',
	data(){
		var validateFile = (rule,value,callback) => {
		    if (value.length < 1) {
		    	callback(new Error('<%=rb.getString("QingXianXuanZeWenJian")%>'))
		    }
		    var pathSplit = value.split(/\\/);
		    var filename = pathSplit[pathSplit.length - 1];
		    if (filename.length > 100) {
		    	callback(new Error('<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>'))
		    }
		    if(filename.length>=1 && filename.length<=100){
		    	if(fileFormatMatch(value,"xlsx,xls,csv")){
		    		callback();
				}else{
					callback(new Error('<%=rb.getString("DaoRuWenJianGeShi")%>'))
				}
		    }
		}
		return {
			options:[{
				value:'0',
				label:'<%=rb.getString("ZhuiJia")%>'
			},
			{
				value:'1',
				label:'<%=rb.getString("FuGai")%>'
			}],
			ruleForm:{
				type:'0',
				file:''
			},
			rules:{
				file:[
					{validator:validateFile,trigger:'change'}
				]
			},
			dataRow:[]
		}
	},
	methods:{
		importFile(){
			$('#strategy_configPlanFile').click();
		},
		/**
		 * 确定按钮
		 * @param ruleFrom:传入的文件模板
		*/
		submit(ruleForm){ 
			var vm = this;
			vm.$refs.ruleForm.validate((valid) => {
				if(valid){
					var files = document.getElementById("strategy_configPlanFile").files;
					var selectVal = this.ruleForm.type;
					$("#configPlanImportForm [name=fileSize]").val(files[0].size);
		    		$("#configPlanImportForm [name=operType]").val(selectVal);
					if(selectVal == 1){
						if(vm.dataRow.length > 0){
							vm.$confirm('<%=rb.getString("QueRenFuGai")%>','<%=rb.getString("QueRen")%>',{
								confirmButtonText:'<%=rb.getString("QueDing")%>',
								cancelButtonText:'<%=rb.getString("QuXiao")%>',
								type:'warning'
							}).then(()=>{
								vm.saveImport()
							}).catch(()=>{
								
							})
						}else{
							vm.saveImport()
						}
					}else{
						vm.saveImport();
					}
				}
			})
		},
		saveImport(){ // 保存表
			var vm = this;
			$(".el-slide").addClass("loading");
			<%-- $("#strategyConfigPlanForm").form('submit', {
			     url: "${ctx}/task/BatchConfiguration/importBatchConfigurationInfos.action",
			     success: function (data) {
			    	 $(".el-slide").removeClass("loading");
			     	if(typeof data == 'string') {
			     		var data = JSON.parse(data);
			     	}
			     	if(data["success"]){
			     		setTimeout(function(){
			     			$(".el-slide").removeClass("loading");
			     			eventBus.$emit('hander-close');
			     		},1000)
			     	}else{
			     		if(data["msg"] == "1"){
			     			$(".el-slide").removeClass("loading");
			     			vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
		        		}else if(data["msg"] == "2"){
		        			$(".el-slide").removeClass("loading");
		        			eventBus.$emit('hander-close');
		        			$("#failureTextBatch").text("<%=rb.getString("BuFenShuJuCuoWuQingChongXingBianJi")%>");
							$("#winDowloadFailureFileBatch").window("setTitle", " <%=rb.getString("XinXi")%>");
							$("#winDowloadFailureFileBatch").window("open");
		        		}else{
		        			$(".el-slide").removeClass("loading");
		        			vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
		        		}
			     	}
			     }
			}) --%>
			uploadWithProgress({
				url: "${ctx}/task/BatchConfiguration/importBatchConfigurationInfos.action",
				form: document.querySelector("#strategyConfigPlanForm"),
				success: function (data) {
		    	 	$(".el-slide").removeClass("loading");
			     	if(data["success"]){
			     		setTimeout(function(){
			     			$(".el-slide").removeClass("loading");
			     			eventBus.$emit('hander-close');
			     		},1000)
			     	}else{
			     		if(data["msg"] == "1"){
			     			$(".el-slide").removeClass("loading");
			     			vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
		        		}else if(data["msg"] == "2"){
		        			$(".el-slide").removeClass("loading");
		        			eventBus.$emit('hander-close');
		        			$("#failureTextBatch").text("<%=rb.getString("BuFenShuJuCuoWuQingChongXingBianJi")%>");
							$("#winDowloadFailureFileBatch").window("setTitle", " <%=rb.getString("XinXi")%>");
							$("#winDowloadFailureFileBatch").window("open");
		        		}else{
		        			$(".el-slide").removeClass("loading");
		        			vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
		        		}
			     	}
			     }
			});
		},
		exportTemp(type){ // 导出按钮
		    exportByForm("${ctx}/task/BatchConfiguration/importBatchConfigurationTemp.action",{
				type: type
			})
		},
		getData(rows){
			this.dataRow = rows;
		},
		close() {
			eventBus.$emit('hander-close');
		}
	},
	mounted(){
		var vm = this;
		$("#strategy_configPlanFile").bind("change", function() {
	    	vm.ruleForm.file = this.value;
		});
		eventBus.$off('hander-ok').$on('hander-ok',this.submit);
		eventBus.$off('hander-export').$on('hander-export',this.exportTemp);
		eventBus.$off('hander-rows').$on('hander-rows',this.getData);
	}
})
</script>