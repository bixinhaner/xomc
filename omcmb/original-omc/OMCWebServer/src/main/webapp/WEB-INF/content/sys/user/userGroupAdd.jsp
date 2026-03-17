<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	#addGroupDiv .basicItem .el-input__inner{
		width:300px;
	}
	#addGroupDiv .roleLabel,#addGroupDiv .userLabel{
		font-size:14px;
	}
	#addGroupDiv .roleLabel:before{
		content:'*';
		color:#f56c6c;
		margin-right:4px;
	}
	#addGroupDiv .el-textarea__inner{
		width:700px;
	}
	#addGroupDiv .basicItem .el-input{
		width:100%;
	}
	#addGroupDiv .addGroupForm{
		margin-left:80px;
	}
	#addGroupDiv .modifyGroupForm{
		margin-left:35px;
	}
</style>
<div id='addGroupDiv'>
	<el-form ref='ruleForm' :model='ruleForm' :rules='rules' :class='groupClass'>
		<el-form-item v-if='addFlag' label='<%=rb.getString("ZuMing")%>' class='basicItem' prop='group_name' style='margin-bottom:20px;'>
			<el-input v-model='ruleForm.group_name'></el-input>
		</el-form-item>
		<div v-else style='margin-top:20px;font-size:14px;color:#999;margin-bottom:20px;'><%=rb.getString("ZuMing")%>:<span style='color:#333;'>{{groupName}}</span></div>
		<div v-if='addFlag'>
			<div style='width:80%;height:350px;margin-top:5px;'>
				<el-pairgrid :id="'selectRole'" :rownumber="true" ref="rolegrid" @selection-change='selectChangeRole' @right-load-success="loadSuccessRole"  :right-url="rightUrlRole" :left-url="leftUrlRole" :width='width' :height="height" row-key="role_id" query-name='role_name' :query-params="roleParams" :title="deviceTitleRole" :messages="{placeholder:'<%=rb.getString("JueSeMingCheng")%>'}">
					<template slot="left">
						<el-table-column type='selection' width="45"></el-table-column>
						<el-table-column prop='role_name' label='<%=rb.getString("JueSeMingCheng")%>' ></el-table-column>
						<el-table-column prop='group_name' label='<%=rb.getString("ZuMing")%>' ></el-table-column>
					</template>
					<template slot="toolbar">
						<div class='queryGroup'>
							<el-input v-model='roleForm.role_name' @keyup.enter.native="queryRole" class='pairgrid-query' placeholder='<%=rb.getString("JueSeMingCheng")%>'></el-input>
							<i @click='queryRole' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
						</div>
					</template>
					<template slot='right'>
						<el-table-column prop='role_name' label='<%=rb.getString("JueSeMingCheng")%>'></el-table-column>
						<el-table-column prop='group_name' label='<%=rb.getString("ZuMing")%>'></el-table-column>
					</template>
				</el-pairgrid>
			</div>
			<el-form-item prop='roleIds'>
				<el-input v-model='ruleForm.roleIds' v-show=false></el-input>
			</el-form-item>
		</div>
		<div v-else>
			<div style='width:80%;height:300px;margin-top:5px;'>
				<el-ctable ref='roleTableModify' :url='modifyGroupUrl' @load-success="loadSuccessModify" :width='width' :height="height"  :query-params="modifyGroupParams" page-size="20" pagination="true" row-key="user_name">
					<template slot="toolbar">
						<div class='queryGroup'>
							<el-input v-model='modifyGroupForm.role_name' @keyup.enter.native="queryRoleModify" class='pairgrid-query' placeholder='<%=rb.getString("JueSeMingCheng")%>'></el-input>
							<i @click='queryRoleModify' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
						</div>
					</template>
					<el-table-column prop='role_name' label='<%=rb.getString("JueSeMingCheng")%>'></el-table-column>
					<el-table-column prop='group_name' label='<%=rb.getString("ZuMing")%>' show-overflow-tooltip="true"></el-table-column>
				</el-ctable>
			</div>
			<el-form-item prop='roleIds'>
				<el-input v-model='ruleForm.roleIds' v-show=false></el-input>
			</el-form-item>
		</div>
		<div style='margin-top:30px;' v-show="cloudFlag =='false'">
			<div style='width:80%;height:350px;margin-top:5px;'>
				<el-pairgrid :id="'selectRole'" :rownumber="true" ref="usergrid" @selection-change='selectChangeUser' @right-load-success="loadSuccessUser"  :right-url="rightUrlUser" :left-url="leftUrlUser" :width='width' :height="height" row-key="user_id" query-name='user_code' :query-params="userParams" :title="deviceTitleUser" :messages="{placeholder:'<%=rb.getString("YongHuMingCheng")%>'}">
					<template slot="left">
						<el-table-column type='selection' width="45"></el-table-column>
						<el-table-column prop='user_code' label='<%=rb.getString("YongHuMingCheng")%>'></el-table-column>
						<el-table-column prop='group_name' label='<%=rb.getString("ZuMing")%>'></el-table-column>
					</template>
					<template slot="toolbar">
						<div class='queryGroup'>
							<el-input v-model='userForm.user_name' @keyup.enter.native="queryUser" class='pairgrid-query' placeholder='<%=rb.getString("YongHuMingCheng")%>'></el-input>
							<i @click='queryUser' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
						</div>
					</template>
					<template slot='right'>
						<el-table-column prop='user_code' label='<%=rb.getString("YongHuMingCheng")%>'></el-table-column>
						<el-table-column prop='group_name' label='<%=rb.getString("ZuMing")%>'></el-table-column>
					</template>
				</el-pairgrid>
			</div>
			<el-form-item prop='userIds'>
				<el-input v-model='ruleForm.userIds' v-show=false></el-input>
			</el-form-item>
		</div>
		<el-form-item v-if='addFlag' label='<%=rb.getString("MiaoShu")%>' prop="group_desc" style='margin-top:20px'>
			<el-input type='textarea' :rows='4' v-model='ruleForm.group_desc'></el-input>
		</el-form-item>
		<div v-else style='font-size:14px;color:#999;'><%=rb.getString("MiaoShu")%>:<span style='color:#333;margin-left:10px;'>{{groupDesc}}</span></div>
	</el-form>
</div>
<script>
	var groupVue = new Vue({
		el:'#addGroupDiv',
		data(){
			var vm = this;
			//校验用户组名格式
			var validateGroupName = (rule,value,callback) => {
				if(value.trim() == vm.defaultGroupName){
					callback();
				}else{
					axios.post('${ctx}/sys/usergroup/checkUserGroupName.action',stringify({
						groupName:vm.ruleForm.group_name.trim()
					})).then(function(response){
						var data = response.data;
						if(!data["success"]){
							callback(new Error('<%=rb.getString("YongHuZuMingChengYiCunZai")%>'))
						}else{
							callback();
						}
					}).catch(function(error){
						callback()
					})
				}
			};
			//校验是否勾选角色
			var validateRole = (rule,value,callback) => {
				if(userVue.groupRowData.built_in == "1" || userVue.groupRowData.built_in == "2"){//内置用户组
					callback()
				}else{
					if(vm.ruleForm.roleIds == ''){
						callback('<%=rb.getString("QingXuanZeJueSe")%>')
					}else{
						callback();
					}
				}
			};
			return{
				cloudFlag:isCloudCore,
				ruleForm:{
					group_name:'',
					roleIds:'',
					userIds:'',
					group_desc:''
				},
				rules:{
					group_name:[
						{validator:validateGroupName,trigger:'blur'},
						{required:true,message:'<%=rb.getString("QingShuRuYongHuZuMingCheng")%>',trigger:'blur'}
					],
					roleIds:[
						{validator:validateRole}
					]
				},
				defaultGroupName:'',
				height:'100%',
				width:'50%',
				leftUrlRole:'${ctx}/sys/usergroup/getRoleList.action',
				rightUrlRole:'',
				leftUrlUser:'${ctx}/sys/usergroup/getUserListForGroup.action',
				rightUrlUser:'',
				roleParams:{
					role_name:'',
					type:'modify',
					operator_code:operator_code
				},
				roleForm:{
					role_name:''
				},
				userParams:{
					user_name:'',
					operator_code:operator_code,
					type:'modify'
				},
				userForm:{
					user_name:''
				},
				deviceTitleRole:['<%=rb.getString("JueSeJiLieBiao")%>','<%=rb.getString("YiXuanJueSe")%>'],
				deviceTitleUser:['<%=rb.getString("YongHuLieBiao")%>','<%=rb.getString("YiXuanYongHu")%>'],
				selectionRole:[],
				selectionUser:[],
				modifyGroupUrl:'${ctx}/sys/usergroup/getRoleList.action',
				modifyGroupParams:{
					group_id:userVue.groupRowData.group_id,
					operator_code:operator_code,
					role_name:'',
					type:'view'
				},
				modifyGroupForm:{
					role_name:''
				},
				addFlag:true,
				groupClass:'',
				groupName:'',
				groupDesc:'',
				defaultRoleIds:[],
				defaultUserIds:[]
			}
		},
		methods:{
			init(){
				var vm = this
				if("${type}" == 'add'){
					vm.addFlag = true;
					vm.groupClass='addGroupForm'
				}
				if("${type}" == 'modify'){
					if(userVue.groupRowData.built_in == "1" || userVue.groupRowData.built_in == "2"){//内置用户组
						vm.addFlag = false;
						vm.groupClass='modifyGroupForm';
						vm.groupName = userVue.groupRowData.group_name;
						vm.groupDesc = userVue.groupRowData.group_desc;
					}else{
						vm.addFlag = true;
						vm.groupClass='modifyGroupForm'
						vm.ruleForm.group_name = userVue.groupRowData.group_name;
						vm.ruleForm.group_desc = userVue.groupRowData.group_desc;
						vm.defaultGroupName = userVue.groupRowData.group_name;
					}
					vm.rightUrlRole = '${ctx}/sys/usergroup/getRoleList.action?group_id='+userVue.groupRowData.group_id+'&operator_code='
							+operator_code + '&type=view'
					vm.rightUrlUser = '${ctx}/sys/usergroup/getUserListForGroup.action?group_id='+userVue.groupRowData.group_id+'&operator_code='
					+operator_code + '&type=view'
					initForm(vm.$refs.ruleForm);
				}
			},
			/**
			 * 勾选角色复选框
			 * param selection {array} 勾选项
			*/
			selectChangeRole(selection){
				this.selectionRole = selection;
			},
			/**
			 * 勾选角色复选框
			 * param selection {array} 勾选项
			*/
			selectChangeUser(selection){
				this.selectionUser = selection;
			},
			//角色列表模糊查询
			queryRole(){
				var vm = this;
				Object.assign(vm.roleParams, vm.roleForm);
			},
			//用户列表模糊查询
			queryUser(){
				var vm = this;
				Object.assign(vm.userParams, vm.userForm);
			},
			//没有勾选框的角色列表模糊查询
			queryRoleModify(){
				var vm = this;
				Object.assign(vm.modifyGroupParams, vm.modifyGroupForm);
			},
			//角色列表加载完成方法
			loadSuccessRole(){
				var vm = this;
				if("${type}" == 'modify'){
					if(userVue.groupRowData.built_in != "1" && userVue.groupRowData.built_in != "2"){//内置用户组
						var data = vm.$refs.rolegrid.getData();
						vm.defaultRoleIds = data.map(function(item){
							return item.role_id
						})
						vm.ruleForm.roleIds = vm.defaultRoleIds.toString();
					}
					initForm(vm.$refs.ruleForm);
				}
			},
			//用户列表加载完成方法
			loadSuccessUser(){
				var vm = this;
				if("${type}" == 'modify'){
					var userData = vm.$refs.usergrid.getData();
					vm.defaultUserIds = userData.map(function(item){
						return item.user_id
					})
					vm.ruleForm.userIds = vm.defaultUserIds.toString();
					initForm(vm.$refs.ruleForm);
				}
			},
			//没有勾选框的角色列表列表加载完成方法
			loadSuccessModify(){
				var vm = this;
				if("${type}" == 'modify'){
					var data = vm.$refs.roleTableModify.getData();
					var roleIds = '';
					data.map(function(item){
						roleIds += item.role_id + ',';
					})
					vm.ruleForm.roleIds = roleIds.substring(0,roleIds.length-1);
				}
				initForm(vm.$refs.ruleForm);
			},
			//新建/修改用户组提交方法
			submit(){
				var vm = this;
				vm.$refs.ruleForm.validate((valid) => {
					if(valid){
						var tipStr = '<%=rb.getString("WuCanShuBianHua")%>';
						if(isFormChanged(vm.$refs.ruleForm)){
							if("${type}" == 'add'){
								var params = {
										type:'${type}',
										group_name:vm.ruleForm.group_name,
										group_desc:vm.ruleForm.group_desc,
										role_ids:vm.ruleForm.roleIds,
										user_ids:vm.ruleForm.userIds,
										operator_code:operator_code
								}
								var message='<%=rb.getString("ChengGong")%>';
							}else{
								var userArr = [];
								var userData = vm.$refs.usergrid.getData().map(function(item){
									return item.user_id
								});
								vm.defaultUserIds.map(function(item){
									if(userData.indexOf(item) == -1){
										userArr.push(item)
									}
								})
								if(userVue.groupRowData.built_in == 1 || userVue.groupRowData.built_in == 2){
									var params = {
											type:'${type}',
											group_name:userVue.groupRowData.group_name,
											group_desc:userVue.groupRowData.group_desc,
											group_id:userVue.groupRowData.group_id,
											role_ids:vm.ruleForm.roleIds,
											user_ids:vm.ruleForm.userIds,
											operator_code:operator_code
									}
									params.uncheck_user_ids = userArr.toString();
								}else{
									var params = {
											type:'${type}',
											group_name:vm.ruleForm.group_name,
											group_desc:vm.ruleForm.group_desc,
											group_id:userVue.groupRowData.group_id,
											role_ids:vm.ruleForm.roleIds,
											user_ids:vm.ruleForm.userIds,
											operator_code:operator_code
									}
									var roleArr = [];
									var roleData = vm.$refs.rolegrid.getData().map(function(item){
										return item.role_id
									});
									vm.defaultRoleIds.map(function(item){
										if(roleData.indexOf(item) == -1){
											roleArr.push(item)
										}
									})
									params.uncheck_role_ids = roleArr.toString();
									params.uncheck_user_ids = userArr.toString();
									
								}
								var message='<%=rb.getString("ChengGong")%>';
							}
							axios.post("${ctx}/sys/usergroup/saveForUI.action",stringify(params)).then(function(response){
								var data = response.data;
								if(data["success"]){
									vm.$message({
			    						message:message,
			    						type:'success',
			    					})
									userVue.$refs.groupTable.refresh();
                                    userVue.$refs.slide.hide();
                                    if("${type}" == 'add'){
                                        userVue.showAddButton = true;
                                    }
								}else{
									vm.$message.error(data["message"])
								}
							})
						}else{
							vm.$message(tipStr)
						}
					}
				})
			},
			//取消新建/修改用户组页面方法
			cancel(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
				if(isFormChanged(vm.$refs.ruleForm)){
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
			}
		},
		watch:{
			selectionRole(){
				var data = this.$refs.rolegrid.getData();
				var roleIds = '';
				if(data.length != 0){
					data.map(function(item){
						roleIds += item.role_id + ","
					})
				}
				this.ruleForm.roleIds = roleIds.substring(0,roleIds.length-1)
			},
			selectionUser(){
				var data = this.$refs.usergrid.getData();
				var userIds = '';
				if(data.length != 0){
					data.map(function(item){
						userIds += item.user_id + ","
					})
				}
				this.ruleForm.userIds = userIds.substring(0,userIds.length-1)
			},
		},
		mounted(){
			this.init();
			eventBus.$off('save-group').$on('save-group',this.submit)
			eventBus.$off('cancel-group').$on('cancel-group',this.cancel)
		}
	})
</script>