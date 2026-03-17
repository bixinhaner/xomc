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
<div id="enbUpgradePage" class="borderPage">
	<el-form ref="form" style="padding:30px 20px;" :model="upgradeform" :rules="rules">
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
				<el-radio-group v-model="upgradeform.type">
					<el-radio label="1" class="CODE_ENB_UPGRADE_IMAGE hidden"><%=rb.getString("RuanJianShengJi")%></el-radio>
					<el-radio label="4" v-if="!onlyHasIMG" class="CODE_ENB_UPGRADE_PATCH hidden"><%=rb.getString("CAZhengShuShengJi")%></el-radio>
					<el-radio label="6" v-if="!onlyHasIMG" class="CODE_ENB_UPGRADE_FPGA hidden"><%=rb.getString("FPGAShengJi")%></el-radio>
					<el-radio label="7" v-if="isTurbo"><%=rb.getString("APShengJi")%></el-radio>
				</el-radio-group>
			</el-form-item>
		</div>
		<div class="form-info">
			<div class="item-title"><%=rb.getString("WenJianLieBiao") %></div>
			<el-form-item prop='rawMode' label="<%=rb.getString("BaoLiuPeiZhi") %>" label-width="150px">
				<el-checkbox v-model="upgradeform.rawMode" style='margin-top:3px;'></el-checkbox>
			</el-form-item>
			<el-form-item prop='fileId' label="<%=rb.getString("ShengJiBanBen") %>" label-width="150px">
				<el-select v-model='upgradeform.fileId'>
					<el-option v-for="item in fileList" :label="item.name" :value="item.value" :key="item.value"></el-option>
				</el-select>
			</el-form-item>
		</div>
		<div class="form-info">
			<el-button @click="submit" type="primary"><%=rb.getString("LiJiShengJi") %></el-button> <span v-if="false" class="linkStyle" style="margin-left:10px;">View Upgrade List</span>
		</div>
		
	</el-form>
</div>
<script>
var enbUpgradePage = new Vue({
	el: '#enbUpgradePage',
	data() {
		var validateVersion = (rule,value,callback) => {
			var vm = this;
			if(value === ''){
				callback(new Error('<%=rb.getString("QingXianXuanZeWenJian")%>'))
			}else{
				vm.fileData.map(function(item){
					if(item.id == value){
						vm.upgradeform.fileId = value;
					}
				})
				callback();
			}
		};
		return {
            enbSelectedRow:{},
			currentVersion:"",
			cellCode:"",
			upgradeform:{
				selectAll:false,
				cellCodes:'',
				euSerialNumber:'',
				ruSerialNumber:'',
				timeZone:'',
				taskname:'',
				status:'',
				fileId:'',
				type:'1',
				rawMode:true,
				productValue:'',
				maxConcurrentNumber:'20',
			},
			rules: {
				fileId:[
					{validator:validateVersion,trigger:'change'}
				],
			},
			fileParams:{
				timeZone:timeZone,
				productValue:"",
				isShowSlave:false,
				page:1,
				rows:50,
				sort:"",
				order:""
			},
			fileUrl:'',
			fileList:[],
			
			product:''
			
		};
	},
	computed: {
		isCloud() {
			return isCloud == 'true';
		},
		isSuperAdmin() {
			return is_super_user == 'true';
		},
		onlyHasIMG() { 
			var vm = this;
			// 非RTS和RTD
			return !['FAP','FAP/\\w+(BS81)\\w+/(DC|SC)'].includes(vm.product);
		},
		isTurbo() {
			var vm = this;
			// Turbo基站 -- 通过全局变量判断
			return isLWAEnable == true;
		},
	},
	watch:{
		"upgradeform.type":function(val){
			var vm = this,
				reg = new RegExp('\\\\',"g");
				vm.fileUrl = "";
				var urlObj = {
						"1" : "${ctx}/cell/version/queryfileInfosList.action",
						"4" : "${ctx}/cell/version/queryfileInfosList.action?file_type=1",
						"6" : "${ctx}/cell/version/queryfileInfosList.action?file_type=6",
						"7" : "${ctx}/cell/version/queryfileInfosList.action?file_type=11"
				}
				vm.$nextTick(function(){
					vm.fileUrl = urlObj[val];
					vm.fileListData(vm.fileUrl);
				})
		},
	},
	methods: {
		init(row,code,sn,status,version,product){
			var vm = this, reg = new RegExp('\\\\',"g");
            vm.enbSelectedRow = row;
			vm.currentVersion = version;
			vm.cellCode = code;
			vm.upgradeform.taskname = "Software Upgrade_" + user_code +"_" + gloableTime;
			vm.fileUrl = '${ctx}/cell/version/queryfileInfosList.action';
			product = product == 'CR-B4860' ? 'CR-B4860/BU' : product;
			
			axios.post('${ctx}/task/upgrade/getProductType.action').then(function(response){
				var productTypeList = response.data;
			
				var productValue ='';
				vm.$nextTick(function(){
					productTypeList.map(function(item){
						if(item.name == product){
							productValue = item.value;
						}
					})
					vm.product = productValue;
					vm.fileParams.productValue = productValue.includes("CR-B4860") ? productValue.substr(0,8) : productValue.replace(reg,'');
					if(productValue.includes("CR-B4860")){
						vm.fileParams.file_type = "10"
					}
					vm.fileListData(vm.fileUrl);
				});
			}).catch(function(error){})
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
		submit(){
	    	var vm = this, reg = new RegExp('\\\\',"g");
	    	var message = '<%=rb.getString("ChengGong")%>';
	    	vm.$refs.form.validate((valid) => {
	    		if(valid){
	    			var params = {};
	    			
	    			params.selectAll = "false";
	    			params.cellCodes = vm.cellCode;
	    			params.euSerialNumber = "";
					params.ruSerialNumber = "";
					
	    			params.timeZone = timeZone;
	    			params.taskName = vm.upgradeform.taskname;
	    			params.status = "active";
	    			params.fileId = vm.upgradeform.fileId;
	    			
				
	    			if(vm.product.includes("CR-B4860")){
						params.nxpUpgradeFlag = '0';
					}
					params.productValue = vm.product;
	    			params.taskType = vm.upgradeform.type;
	    			params.rawMode = vm.upgradeform.rawMode == true ? 'false' : 'true';
	    			params.maxConcurrentNumber = '20';
    				url = '${ctx}/task/upgrade/addTask.action'
  	    			vm.$confirm("<%=rb.getString("QueRenXinJianRenWu")%>",'<%=rb.getString("QueRen")%>',{
  						customClass:'warningConfirm',
  						confirmButtonText:'<%=rb.getString("QueDing")%>',
  						cancelButtonText:'<%=rb.getString("QuXiao")%>',
  						type:'warning',
  						closeOnClickModal:false
  					}).then(() => {
						var ckParams = {
								currentVersion: vm.currentVersion,
								targetVersion: vm.upgradeform.fileId
							};
						
						axios.post('${ctx}/task/upgrade/validateUpgradeVersion.action',stringify(ckParams)).then(function(response){
							var data = response.data;
							if(data["success"] == true) { // Âú×ãÖÐ¼ä°æ±¾Éý¼¶
								vm.$message.error(data["message"]);
							}else{
  								vm.saveTask(url,params,message);
							}
						})
  					}).catch(() => {})
	    			
	    		}else{
	    			return false;
	    		}
	    	})
		},
		saveTask(url,params,message){
			var vm = this;
			axios.post(url,stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$message({
						message:message,
						type:'success',
					})
				}else{
					vm.$message.error(data["message"])
				}
			})
		},
	},
	mounted() {
		eventBus.$off("enb-data").$on("enb-data",this.init)
	}
});

</script>
