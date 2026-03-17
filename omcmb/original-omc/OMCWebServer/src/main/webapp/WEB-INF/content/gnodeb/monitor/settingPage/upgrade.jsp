<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<style>
.rightWarpLayer .el-form-item__label{
	font-size: 12px;
	color:#7A7992; 
}
</style>
<div id="gnbSettingUpgradePages" class='rightWarpLayer commonWarp' style='background: #FFFFFF;width: calc(100% - 2px); height: 100%;'>
	<el-form :model='ruleForm' ref="ruleForm" :rules="rules" style='padding:30px 20px;' label-position="left" label-width="180px" class='rightWarpLayerContent'>
		<div style='margin-bottom: 10px;'>
			<span class='commonTemplateText12' style='display: inline-block; width: 180px;'><%=rb.getString("DangQianBanBen")%></span>
			<span class='commonGeneral12' v-html='softwareVersion'></span>
		</div>
		<el-form-item prop='rawMode' label='<%=rb.getString("BaoLiuPeiZhi") %>'>
			<el-checkbox v-model="ruleForm.rawMode"  true-label="false" false-label="true" style='margin-top:3px;'></el-checkbox>
		</el-form-item>
		<el-form-item prop="fileId" label='<%=rb.getString("ShengJiBanBen")%>'>
			<el-select v-model="ruleForm.fileId" style="padding-top: 7px;">
				<el-option v-for="item in versionList" :label="item.version" :value="item.id" :key="item.id"></el-option>
			</el-select>
		</el-form-item>
	</el-form>
	<div v-if="gnbOptShow" class='commonFlex commonBorderTop commonFormFotter'>
		<div>
			<el-button type="primary" @click="submit"><%=rb.getString("LiJiShengJi")%></el-button>
		</div>
	</div>
</div>

<script type="text/javascript">
	var upgradePagesVue = new Vue({
	    el: '#gnbSettingUpgradePages',
	    data() {
	    	var vm = this,
	    		validateVersion = (rule,value,callback) => {
	    			if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("QingXianXuanZeWenJian")%>'))
					}else{
						callback();
					}
				};
	    	return {
	    		ruleForm:{
	    			taskName: '',
	    			fileId:'',
					rawMode:'false'
				},
				rules:{					
					fileId:[
						{validator: validateVersion,trigger:'change'}
					]
				},
				buttonGroups:[
					{value:'(FAP/\\w+BU2210)|^001$|^003$|^5G BBU-F1$',text:'BaiBNX'},
					{value:'FAP/\\w*BSC\\w+',text:'BaiBNQ'},
				],
				versionList: [],
				small_cell_code: '',
				product: '',
				softwareVersion: ''
	    	}
	    },
		computed: {
			gnbOptShow() {
				return writableMap.CODE_GNB_UPGRADE_IMAGE == true;
			},
		},
	    methods: {
	    	upgradeInit(){
				var vm = this, productValue = '', reg = new RegExp('\\\\',"g"), str = Math.random().toString();
				
				vm.small_cell_code = gnbTabSettingVue.rowData.small_cell_code;
				vm.softwareVersion = gnbTabSettingVue.rowData.software_version; // 当前版本
				vm.product = gnbTabSettingVue.rowData.product;
				// 产品型号
    			if(vm.product == '' || vm.product == null || vm.product == undefined){
    				return;
    			}else if(vm.product == 'BaiBNX'){
    				productValue = '(FAP/\\w+BU2210)|^001$|^003$|^5G BBU-F1$';
    			}else if(vm.product == 'BaiBNQ'){
    				productValue = 'FAP/\\w*BSC\\w+';
    			}
				// 需转译 productValue 值
				productValue = productValue.replace(reg,'');
				//axios.post('${ctx}/cell/version/queryfileInfosList.action?file_type=0',stringify({
				axios.post('${ctx}/cell/version/queryAllFileInfosList.action?file_type=0&rd=' + str,stringify({
					timeZone: timeZone,
					productValue : productValue,
					isShowSlave: false,
					isGnb: 1					
				})).then(function(response){
					var data = response.data;
					
					if(data && data.rows.length > 0){
						vm.versionList = data.rows.map(function(item){
				   			if (item){
				   				return {version:item.version,id:item.id}
				   			}
				   		})
					}else{
						vm.versionList = [];
					}
				}).catch(function(error){})
			},
			submit(){
				var vm = this, curTaskName = '', params = {};
				
					curTaskName = '<%=rb.getString("RuanJianShengJi")%>' + '_' + user_code + '_' + formatDate(new Date(gloableTime)); 
					params.timeZone = timeZone;
					params.isGnb = 1;
	    			params.cellCodes = vm.small_cell_code;// small_cell_code
	    			params.taskName = curTaskName;
	    			
					params.selectAll = 'false';//设备指定 默认值  指定执行-false
	    			params.status = 'active';// 执行方式 默认值 active-立即执行
					params.taskType = '1';// 默认值 1
					
					params.maxConcurrentNumber = '20';//设备并发数 默认值 20
					params.rawMode = vm.ruleForm.rawMode;
	    			params.fileId = vm.ruleForm.fileId; //版本对应参数
	    			// 产品型号
	    			if(vm.product == 'BaiBNX'){
	    				params.productValue = '(FAP/\\w+BU2210)|^001$|^003$|^5G BBU-F1$';
	    			}else if(vm.product == 'BaiBNQ'){
	    				params.productValue = 'FAP/\\w*BSC\\w+';
	    			}
										
				//校验名称是否存在
				vm.$refs.ruleForm.validate((valid) => {
		    		if(valid){
		    			axios.post('${ctx}/task/upgrade/taskNameExist.action',stringify({
							taskName: curTaskName.trim(),
							isGnb:1,
							taskType: 1
						})).then(function(response){
							var data = response.data;
							if(data["success"]){
								if(data["message"] == "true"){
									vm.$message.error('<%=rb.getString("RenWuMingChengYiCunZai")%>');
								}else{
									vm.$confirm('<%=rb.getString("QueRenXinJianRenWu")%>','<%=rb.getString("QueRen")%>',{
				  						customClass:'warningConfirm',
				  						confirmButtonText:'<%=rb.getString("QueDing")%>',
				  						cancelButtonText:'<%=rb.getString("QuXiao")%>',
				  						type:'warning',
				  						closeOnClickModal:false
				  					}).then(() => {
				  						axios.post('${ctx}/task/upgrade/addTask.action',stringify(params)).then(function(response){
						    				var data = response.data;
						    				if(data["success"]){
						    					vm.$message({
						    						type:'success',
						    						message:'<%=rb.getString("ChengGong")%>'
						    					})
						    				}else{
						    					vm.$message.error(data["message"])
						    				}
						    			}).catch(function(error){})
				  					}).catch(() => {})
								}
							}
						}).catch(function(error){ })
		    			
		    		}else{
		    			return false;
		    		}
		    	})
			},
			
	    },
		mounted(){
	    	this.upgradeInit();
	    }
	});
	
</script> 