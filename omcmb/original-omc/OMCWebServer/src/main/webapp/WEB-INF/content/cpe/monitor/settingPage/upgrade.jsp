<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
.borderPage {
	border:1px solid #d5dcec;
	border-radius:10px;
	height:100%;
	background:#fff;
}
.form-info {
	margin:10px 20px;
}
.form-info .info-title {
	color:#7A7992;
	display:inline-block;
	width:120px;
}
.form-info .el-radio-group .el-radio {
	display:block;
	margin-bottom:20px;
	margin-left:0 !important;
}
.item-title {
	margin:20px 0;
	font-weight:bold;
}
.form-info .el-form-item__label {
	font-size:12px;
	text-align:left;
	line-height:unset;
}
.linkStyle {
	color:#4d84ff;
	font-size:14px;
	text-decoration:underline;
	cursor:pointer;
}
</style>
<div id="setting_cpeUpgradePage" class="borderPage">
	<el-form ref="upgradeform" style="padding:30px 20px;" :model="upgradeform" :rules="rules">
		<div class="form-info" style="margin-bottom:20px;">
			<span class="info-title"><%=rb.getString("DangQianBanBen")%></span><span>{{currentVersion}}</span>
		</div>
		<div v-if="false" class="form-info">
			<span class="info-title"><%=rb.getString("HuiTuiDao")%></span><span>BaiBS_RTS_3.7.10</span>
			<span class="linkStyle" style="margin-left:15px;">Rollback now</span>
		</div>
		<div class="form-info">
			<div class="item-title"><%=rb.getString("ShengJiLeiXing") %></div>
			<el-form-item prop="type">
				<el-radio-group v-model="upgradeform.upgradeMode" @change="modeChange">
					<el-radio border label="image">Image Upgrade</el-radio>
					<el-radio border label="module">Module Upgrade</el-radio>
				</el-radio-group>
			</el-form-item>
		</div>
		<div class="form-info">
			<div class="item-title"><%=rb.getString("WenJianLieBiao") %></div>
			<el-form-item>
				<el-checkbox class="filterFile" 
					v-model="noFilterFile" 
					@change="filterFileChange" 
					label="<%=rb.getString("ZiDongGenJuMoKuaiXingHaoGuoLv")%>">
				</el-checkbox>
	
				<el-checkbox class="filterFile" v-if="upgradeform.upgradeMode == 'image'"
					v-model="upgradeform.needModuleUpgrade" 
					true-label="true"
					false-label="false"
					@change="filterModuleFileChange"
					label="Auto Upgrade related Module version">
				</el-checkbox>
			</el-form-item>
			<el-form-item prop='version' label="<%=rb.getString("ShengJiBanBen") %>" label-width="150px">
				<el-select v-model='upgradeform.version'>
					<el-option v-for="item in fileList" :label="item.name" :value="item.value" :key="item.value"></el-option>
				</el-select>
			</el-form-item>
		</div>
		<div class="form-info">
			<el-button @click="createTask" type="primary">Upgrade Now</el-button> <span v-if="false" class="linkStyle" style="margin-left:10px;">View Upgrade List</span>
		</div>
		
	</el-form>
</div>
<script>
var cpeUpgradePage = new Vue({
	el: '#setting_cpeUpgradePage',
	data() {
		var validateVersion = (rule,value,callback) => {
			var vm = this;
			if(value === ''){
				callback(new Error('<%=rb.getString("QingXianXuanZeWenJian")%>'))
			}else{
				vm.fileData.map(function(item){
					if(item.id == value){
						vm.upgradeform.fileIds = value;
						vm.upgradeform.fileName = item.file_name;
						vm.upgradeform.version = item.version;
						
					}
				})
				callback();
			}
		};
		return {
			currentVersion:'',
			model_name:'',
			module_name:'',
			upgradeform:{
				taskname:'',
				upgradeMode: 'image',
				moduleName: '',
				needModuleUpgrade: 'false',
				cellCodes:'',
				fileIds:'',
				fileName:'',
				status:'active',
				exetime:'',
				deviceAssign:'0',
				version:'',
				connection_status: ''
				
			},
			rules:{
				version:[
					{validator:validateVersion,trigger:'change'}
				],
				
			},
			fileParams:{
				timeZone:timeZone,
				isShowSlave:false,
				fileId:'',
				model_name:'',
				moduleName: '',
				page:1,
				rows:200,
				sort:'',
				order:''
			},
			fileUrl:'',
			fileData:{},
			fileList:[],
			noFilterFile:true,
			
			
		};
	},
	computed: {

	},
	watch:{
		
	},
	methods: {
		init(){
			var vm = this;
			
			vm.currentVersion = sessionStorage.getItem('softVersion');
			vm.model_name = sessionStorage.getItem('model_name');
			vm.module_name = sessionStorage.getItem('module_name');
			vm.fileParams.model_name = vm.model_name;
			vm.upgradeform.cellCodes = sessionStorage.getItem('CPE_CODE');
			vm.upgradeform.taskname = "Software Upgrade_" + user_code +"_" + gloableTime;
			
			vm.fileUrl = '${ctx}/cell/version/queryfileInfosList.action?file_type=9&timeZone='+timeZone;
			
			vm.fileListData(vm.fileUrl);
		},
		fileListData(url){
			var vm = this;
			
			axios.post(url,stringify(vm.fileParams)).then(function(response){
				let data = response.data;
				
				vm.fileData = data.rows;
				vm.fileList = [];
				if(data.rows.length > 0){
					data.rows.map(function(item){
						var obj = {}
						obj.name = item.version;
						obj.value = item.id;
						vm.fileList.push(obj)
					})		
					
				}
				
			}).catch(function(error){})
		},
		// Upgrade Mode 切换事件
		modeChange(val) {
			var vm = this;

			if(vm.upgradeform.upgradeMode == 'image') {
				vm.fileUrl = '${ctx}/cell/version/queryfileInfosList.action?file_type=9&timeZone='+timeZone;
			}else {
				vm.fileUrl = '${ctx}/cell/version/queryModuleFileInfos.action';
			}
			vm.fileListData(vm.fileUrl);
		},
		// 文件过滤开关
		filterFileChange(val){
			var vm = this;

			if(val){
				vm.fileParams.model_name = vm.model_name;
			}else{
				vm.fileParams.model_name = '';
			} 

			vm.fileListData(vm.fileUrl)
		},
		// 文件过滤开关
		filterModuleFileChange(val){
			var vm = this;

			if(val){
				vm.fileParams.module_name = vm.module_name;
			}else{
				vm.fileParams.module_name = '';
			}
		},
		createTask(){
			var vm = this;
			var message = '<%=rb.getString("ChengGong")%>';
	    	vm.$refs.upgradeform.validate((valid) => {
	    		if(valid){
	    			var params = {},urls='';
	    			params.timeZone = timeZone;
	    			params.cellCodes = vm.upgradeform.cellCodes;
	    			params.taskName = vm.upgradeform.taskname;

					params.deviceAssign = vm.upgradeform.deviceAssign;
					params.version = vm.upgradeform.version;
					params.taskType = '1';
					params.rawMode = 'false';
	    			params.status = vm.upgradeform.status;

	    			params.file_id = vm.upgradeform.fileIds;
					params.file_name = vm.upgradeform.fileName;

					params.upgradeMode = vm.upgradeform.upgradeMode;
					params.moduleName = vm.upgradeform.moduleName;
					params.needModuleUpgrade = vm.upgradeform.needModuleUpgrade;

					urls = '${ctx}/task/upgrade/cpe/addTask.action';
					
					axios.post('${ctx}/task/upgrade/cpe/isNeedMoreTimeForUpgrade.action', stringify({
       						cpeCodes:params.cellCodes,
							deviceAssign:vm.upgradeform.deviceAssign,
       						destVersion:params.version
       					})).then(function(response){
       						let data = response.data;
       						
       						if (data["isNeedMoreTime"]) {
       							vm.$confirm( '<%=rb.getString("ShengJiShiJianJiaoChangShiFouJiXu")%>' ,'<%=rb.getString("QueRen")%>',{
       			    				customClass:'warningConfirm',
       								confirmButtonText:'<%=rb.getString("QueDing")%>',
       								cancelButtonText:'<%=rb.getString("QuXiao")%>',
       								type:'warning',
       								closeOnClickModal:false
       							}).then(function(){
       								vm.saveTask(urls,params,message);
       							}).catch()
       						} else {
							   var msg = ''
							   if(vm.productModelDifference){
									msg = '<%=rb.getString("PiPeiXingHaoWenJianQueDing")%>' +'<%=rb.getString("QueRenXinJianRenWu")%>';
							   }else{
								   	msg = '<%=rb.getString("QueRenXinJianRenWu")%>';
							   }
  								vm.$confirm( msg ,'<%=rb.getString("QueRen")%>',{
  				    				customClass:'warningConfirm',
  									confirmButtonText:'<%=rb.getString("QueDing")%>',
  									cancelButtonText:'<%=rb.getString("QuXiao")%>',
  									type:'warning',
  									closeOnClickModal:false
  								}).then(function(){
  									vm.saveTask(urls,params,message);	
  								}).catch()
       						}
          				}).catch(function(error){
          				
          				})
					
	    			
	    		}else{
	    			return false;
	    		}
	    	})
		},
		saveTask(urls,params,message){
			var vm = this;
			axios.post(urls,stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$message({
						message:message,
						type:'success',
					})
				}else{
					vm.$message.error(data["message"])
				}
			}).catch(function(error){})
			
		},
	},
	mounted() {
		this.init();
	}
});

</script>
