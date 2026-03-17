<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#addRoleDiv .basicItem .el-input__inner{
		width:300px;
	}
	
	#addRoleDiv .el-textarea__inner{
		width:700px;
	}
	#addRoleDiv .el-input{
		width:100%;
	}
	#addRoleDiv .el-tree{
		height:100%;
		overflow:auto;
	}
	#addRoleDiv .treeLabel{
		font-size:14px;
		color:#333;
	}
	#addRoleDiv .addRoleForm{
		margin-left:80px;
	}
	#addRoleDiv .modifyRoleForm{
		margin-left:35px;
	}
	/*.featureContent{
		 width:1000px;
		 height:540px;
		 border:1px solid #DEDFE6;
		 margin-top:5px;
		 display:flex;
		 flex-direction:column
	}*/
</style>
<div id='addRoleDiv'>
	<el-form ref='roleForm' :model='roleForm' :rules='rules' :class='roleClass'>
		<el-form-item label='<%=rb.getString("JueSeMingCheng")%>' class='basicItem' prop='role_name'>
			<el-input v-model='roleForm.role_name' :disabled="viewRoleFlag"></el-input>
		</el-form-item>
		<el-form-item prop='batch_operation'>
			<el-checkbox v-model="roleForm.batch_operation" true-label="1" false-label="0" :disabled='viewRoleFlag'><%=rb.getString("ZhiChiPiLiang") %></el-checkbox>
		</el-form-item>
		<div style='margin-top:20px;'>
			<label style='font-size:14px;'><%=rb.getString("GongNengQuanXianLieBiao") %></label>
			<el-ctree ref='featureTree' style="height: 500px;width: 1000px;"
				@node-click="nodeClick"
				:readonly="!showViewFlag"
				title="<%=rb.getString("QuanXianLieBiao")%>"
				placeholder='<%=rb.getString("QuanXianLieBiao")%>'
				:data="featureData"
				:cascade="['wForms > forms']"
				:check-forms="[
					{key: 'forms',label: '<%=rb.getString("ZhiDuQuanXuan")%>',prop: 'checked'},
					{key: 'wForms',label: '<%=rb.getString("KeXieQuanXuan")%>',prop: 'write'}
				]"
				:ignore="featureIgnore"></el-ctree>
		</div>
		<div style='margin-top:30px;'>
			<label style='font-size:14px;'><%=rb.getString("SheBeiZu")%></label>
			<div style='width:1000px;height:320px;border:1px solid #DEDFE6;margin-top:5px;display:flex;flex-direction:column;'>
				<div style='margin-top:10px;'>
					<div class='queryGroup'>
						<el-input v-model='deviceForm.group_name' @keyup.enter.native="queryDevice" class='pairgrid-query' placeholder='<%=rb.getString("SheBeiZuMingCheng")%>'></el-input>
						<i @click='queryDevice' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
				</div>
				<p style='font-size:13px;margin:10px 0px 10px 10px;'><%=rb.getString("SheBeiZuLieBiaoChaKanYiXuanSheBei")%></p>
				<el-ctable ref='deviceTable' style="flex: auto;overflow: auto;"
					@selection-change="changeDevice" :default-checked="defaultChecked" :url='deviceUrl' :id="'deviceGroupTable'" width='100%' height="100%"  :query-params="deviceParams" pagination="true" row-key="group_id">
					<el-table-column v-if='showViewFlag' type='selection' width='55'></el-table-column>
					<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZuMingCheng")%>'></el-table-column>
				</el-ctable>
			</div>
			<el-form-item prop='deviceMsg'>
				<el-input v-model='roleForm.deviceMsg' v-show=false></el-input>
			</el-form-item>
		</div>
		<el-form-item label='<%=rb.getString("MiaoShu")%>' style='margin-top:20px' prop='role_desc'>
			<el-input type='textarea' :rows='4' v-model='roleForm.role_desc' :disabled='viewRoleFlag'></el-input>
		</el-form-item>
	</el-form>
</div>
<script>
	var addRoleVue = new Vue({
		el:'#addRoleDiv',
		data(){
			var vm = this;
			//校验角色名称
			var validateRoleName = (rule,value,callback) => {
				if(value.trim() == vm.defaultRoleName){
					callback();
				}else{
					axios.post('${ctx}/sys/role/checkRoleName.action',stringify({
						role_name:vm.roleForm.role_name.trim()
					})).then(function(response){
						var data = response.data;
						if(data["success"]){//不存在
							callback()
						}else{
							callback(new Error('<%=rb.getString("JueSeJiMingChengYiCunZai")%>'));
						}
					}).catch(function(error){
						callback()
					})
				}
			};
			//校验是否勾选设备组
			var validateMsg = (rule,value,callback) => {
				var data = vm.$refs.deviceTable.getChecked();
				if(data.length == 0){
					callback(new Error('<%=rb.getString("QingXuanZeSheBeiZu")%>'));
				}else{
					callback();
				}
			};
			return{
				roleForm:{
					role_name:'',
					batch_operation:'1',
					role_desc:'',
					deviceMsg:''
				},
				rules:{
					role_name:[
						{validator:validateRoleName,trigger:'blur'},
						{required:true,message:'<%=rb.getString("QingShuRuJueSeMingCheng")%>',trigger:'blur'}
					],
					deviceMsg:[
						{validator:validateMsg}
					]
				},
				defaultRoleName:'',
				deviceParams:{
					type:'modify',
					operator_code:operator_code,
					group_name:''
				},
				deviceForm:{
					group_name:''
				},
				deviceUrl:'${ctx}/sys/role/getDeviceGroup.action',
				roleClass:'',
				defaultChecked:[],
				viewRoleFlag:false,
				viewRoleClass:"${type}" == "view"?"readonly":'',
				showViewFlag:"${type}" == "view"?false:true,
				disabelList:['1'],
				filterText:'',
				featureData:[],
				featureIgnore: {
					forms: ['1','7','10','30','32','38','64','76','75','92','96','97','98','122','10811','10812','10813','10814','10817'],
					wForms: ['1','58','59','60','63','117']
				},
				result: {},
				resultDefault:{
					readonly:[]
				},
				idRelCodeMap: {}
			}
		},
		methods:{
			nodeClick(obj,node,t) {
				var vm = this;
				
				vm.$nextTick(function(){
					vm.getResult();
				});
			},
			//设备组模糊查询
			queryDevice(){
				var vm = this;
				Object.assign(vm.deviceParams,vm.deviceForm);
			},
			/**
			 * 过滤节点方法
			 * param value{string} 当前过滤值
			 *       data{object} 当前节点属性对象
			*/
			filterNode(value,data){
				if(!value) return true;
				return data.text.indexOf(value) !== -1
			},
			//获取权限勾选结果
			getResult(){
				var vm = this,
					res = vm.$refs.featureTree.getResult(),
					rlist = res.forms,
					wlist = res.wForms;

				vm.result = {
					readonly: rlist.sort(function(a,b){return a-b}),
					write: wlist.sort(function(a,b){return a-b})
				};
			},
			init(){
				var vm = this;
				if("${type}" == 'add'){
					vm.roleClass='addRoleForm';
				}else{
					vm.roleClass='modifyRoleForm';
					vm.roleForm.role_name = userVue.roleRowData.role_name;
					vm.roleForm.role_desc = userVue.roleRowData.desc;
					vm.roleForm.batch_operation = userVue.roleRowData.batch_operation;
					vm.defaultRoleName = userVue.roleRowData.role_name;
					if("${type}" == 'view'){
						vm.viewRoleFlag = "true";
						vm.deviceParams = {
							type:'view',
							operator_code:operator_code,
							group_name:'',
							role_id:userVue.roleRowData.id
						}
					}
					initForm(vm.$refs.roleForm);
				}
				if("${type}" == 'modify'){
					var paramsDevice = {
							operator_code:operator_code,
							role_id:userVue.roleRowData.id,
					}
					axios.post("${ctx}/sys/role/getGroupIds.action",stringify(paramsDevice)).then(function(response){
						var data = response.data;
						vm.defaultChecked = data.group_id.split(',').map(Number).sort(function(a,b){return a-b});
					})
				}
				var params = {
						type:'add',
						operator_code:operator_code
				}
				if("${type}" != 'add'){
					params.type = 'modify';
					params.role_id = userVue.roleRowData.id;
				}
				axios.post("${ctx}/sys/role/getFeature.action",stringify(params)).then(function(response){
					var data = vm.handleData(response.data);
					// 初始不显示的节点
					vm.initIgnore(data);
					vm.$nextTick(function(){
						// 设置默认勾选
						if("${type}" != 'add'){
							setTimeout(function(){
								vm.getResult();
								vm.resultDefault = vm.result;
								
							},100)
						}
						vm.featureData = JSON.parse(JSON.stringify(data));
						if("${type}" != 'add'){
							setTimeout(function(){
								vm.$refs.featureTree.reviewForms(vm.featureData);
							},0)
						}
					});
				})
				
			},
			/**
			 * 处理数据
			 * param data {array} 生成树的节点数据
			         type {string} 类型
			*/
			handleData(data,type){
				var vm = this;
				data.map(function(item){
					if(item.pid == '0'){
						item.pid = 'root';
					}
					if(item.children.length >0){
						vm.handleData(item.children,type)
						item.isLeaf = false
					}else{
						item.isLeaf = true;
					}
				})
				return data;
			},
			// 初始化不显示的节点
			initIgnore(data) {
				var vm = this; //featureIgnore
				data.map(function(row){
					if(row.children && row.children.length) vm.initIgnore(row.children);
					
					if(row.reWrite == 'true' || row.reWrite == true) {
						
					}else {
						row.isLeaf && row.id != '94' && vm.featureIgnore.wForms.push(row.id);
					}
				})
			},
			//新建/修改角色提交方法
			submit(){
				var vm = this;
				var changeFlag = vm.checkChange();
				vm.$refs.roleForm.validate((valid) => {
					if(valid){
						if(changeFlag == 'true' || "${type}" == 'add'){
							vm.getResult();

							vm.idRelCodeMap = {};
							vm.getIdMap(vm.featureData);
							
							var feature_ids = [{id: '1',code: 'CODE_DASHBOARD',write: true}];
							vm.result.write.map(function(item){ 
								var code = vm.idRelCodeMap[item];
								feature_ids.push({id:item,code: code,write:true});
							})
							vm.result.readonly.map(function(item){ 
								var code = vm.idRelCodeMap[item];
								if(!vm.result.write.includes(item)){
									feature_ids.push({id:item,code: code,write:false})
								}
							})
							var data = vm.$refs.deviceTable.getChecked();
							var device_group_ids = '';
							data.map(function(item){
								device_group_ids += item + ','
							})
							device_group_ids = device_group_ids.substring(0,device_group_ids.length-1);
							var params = {
								type:"${type}",
								role_name:vm.roleForm.role_name,
								batch_operation:vm.roleForm.batch_operation,
								role_desc:vm.roleForm.role_desc,
								device_group_ids:device_group_ids,
								feature_ids:JSON.stringify(feature_ids),
								operator_code:operator_code
							}
							if("${type}" == 'modify'){
								params.role_id = userVue.roleRowData.id;
							}
							axios.post("${ctx}/sys/role/saveRolesInfoForUI.action",stringify(params)).then(function(response){
								var data = response.data;
								if("${type}" == 'add'){
									var message = '<%=rb.getString("ChengGong")%>';
								}else{
									var message = '<%=rb.getString("ChengGong")%>';
								}
								if(data["success"]){
									vm.$message({
			    						message:message,
			    						type:'success',
			    					})
                                    userVue.$refs.roleTable.refresh();
                                    userVue.$refs.slide.hide();
                                    if("${type}" == 'add'){
                                        userVue.showAddButton = true;
                                    }
								}else{
									vm.$message.error(data["message"])
								}
							})
						}else{
							vm.$message('<%=rb.getString("WuCanShuBianHua")%>');
						}
					}
				})
				
			},
			getIdMap(list) {
				var vm = this;
				
				list.map(function(item) {
					var sub = item.children;
					
					if(sub && sub.length) {
						vm.getIdMap(sub);
					}
					
					vm.idRelCodeMap[item.id] = item.code;
				})
			},
			//取消新建/修改角色页面
			cancel(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>';
				var changeFlag = vm.checkChange();
				if(changeFlag == "true"){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						userVue.$refs.slide.hide();
						if("${type}" == 'add'){
							userVue.showAddButton = true;
						}
					}).catch(() => {
						
					})
				}else{
					userVue.$refs.slide.hide();
					if("${type}" == 'add'){
						userVue.showAddButton = true;
					}
				}
			},
			//校验参数是否有变化
			checkChange(){
				var vm = this;
				vm.getResult();
				if("${type}" == 'add'){
					var readChange = vm.result.readonly.length > 0 //有改变
					var writeChange = vm.result.write.length > 0 //有改变
					var deviceChange = vm.$refs.deviceTable.getChecked().length > 0;
				}else{
					var readChange = vm.result.readonly.toString() != vm.resultDefault.readonly.toString(); //有改变
					var writeChange = vm.result.write.toString()  != vm.resultDefault.write.toString()//有改变
					var deviceChange = vm.$refs.deviceTable.getChecked().map(Number).sort(function(a,b){return a-b}).toString() != vm.defaultChecked.toString();
				}
				
				var treeChange = readChange || writeChange || deviceChange
				if(isFormChanged(vm.$refs.roleForm) || treeChange){
					return "true";
				}else{
					return "false";
				}
			},
			//设备组复选框勾选改变
			changeDevice(){
				var vm = this;
				setTimeout(function(){
					vm.$refs.roleForm.validateField("deviceMsg")
				},10)
			}
		},
		mounted(){
			this.init();
			eventBus.$off('save-role').$on('save-role',this.submit);
			eventBus.$off('cancel-role').$on('cancel-role',this.cancel);
		}
	})
</script>
