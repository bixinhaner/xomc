<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#addUserDiv .el-form-item.is-required:not(.is-no-asterisk)>.el-form-item__label:before{
		content:'*';
		color:#f56c6c;
		margin-right:4px;
	}
	#addUserDiv .basicItem .el-form-item{
		margin-bottom:20px;
	}
	#addUserDiv .el-input__inner{
		width:300px;
	}
	#addUserDiv .basicItem{
		display:inline-block;
		margin-right:98px;
	}
	#addUserDiv .lockItem{
		width:695px;
		height:40px;
		border:1px solid #DCDFE6;
		margin-top:35px;
	}
	#addUserDiv .lockItem .el-radio-group{
		margin-top:12px;
	}
	#addUserDiv .groupItem{
		position:relative;
		margin-top:25px;
	}
	#addUserDiv .groupItem .groupLabel{
		font-size:14px;
	}
	#addUserDiv .groupItem .groupLabel:before{
		content:'*';
		color:#f56c6c;
		margin-right:4px;
	}
	#addUserDiv .groupDiv{
		width:695px;
		height:auto;
		min-height:40px;
		border:1px solid #DCDFE6;
		display:flex;
		margin-top:10px;
	}
	#addUserDiv .groupHide{
		width:695px;
		height:auto;
		min-height:40px;
		position:absolute;
		top:28px;
	}
	#addUserDiv .leftItem{
		display:inline-block;
		height:20px;
		line-height:20px;
		border-radius:10px;
		background:#F7F7F7;
		padding:0 10px;
		margin-bottom:8px;
		margin-left:15px;
		border:1px solid #E8EAEC;
		font-size:12px;
	}
	#addUserDiv .selectGroupItem{
		display:flex;
		position:absolute;
		z-index:10;
		top:72px;
		box-shadow:0 0 10px rgba(0,0,0,0.1);
	}
	#addUserDiv .selectGroupItem .el-checkbox-group{
		display:inline-block;
	}
	#addUserDiv .selectGroupItem .el-radio{
		line-height:normal;
	}
	#addUserDiv .el-checkbox+.el-checkbox{
		margin-left:0px;
	}
	#addUserDiv .dateItem{
		margin-top:30px;
	}
	#addUserDiv .dateItem .dateLabel{
		font-size:14px;
	}
	#addUserDiv .dateItem .dateLabel:before{
		content:'*';
		color:#f56c6c;
		margin-right:4px;
	}
	#addUserDiv .dateItem .el-date-editor.el-input{
		width:300px;
	}
	#addUserDiv .el-textarea__inner{
		width:695px;
	}
	#addUserDiv .el-input{
		width:300px;
	}
	#addUserDiv .el-icon-circle-close:before{
		color:#C0C4CC;
	}
	#addUserDiv .el-icon-down:before{
		color:#C0C4CC;
	}
	#addUserDiv .phoneRequired{
		position:absolute;
		top:7px;
		left:-12px;
		color:#F56C6C;
		font-size:16px;
	}
</style>
<div id='addUserDiv'>
	<el-form ref='ruleForm' :model='ruleForm' :rules='rules' style='margin-left:40px;'>
		<el-form-item class='basicItem' label='<%=rb.getString("YongHuMingCheng")%>' prop='userName'>
			<el-input v-model='ruleForm.userName' :disabled='"${type}" == "modify"?true:false'></el-input>
		</el-form-item>
		<el-form-item class='basicItem' label='<%=rb.getString("YouXiang")%>' prop='email'>
			<el-input v-model='ruleForm.email' :disabled='"${type}" == "modify"?true:false'></el-input>
		</el-form-item>
		<el-form-item class='basicItem' label='<%=rb.getString("ShouJiHao")%>' prop='phone' style='margin-right:0px;'>
			<a class="phoneRequired">*</a>
			<el-input v-model='ruleForm.phone'></el-input>
		</el-form-item>
		<el-form-item label='<%=rb.getString("YouXiaoQiSuoDing")%>' prop='status'>
			<div class='lockItem'>
				<el-radio-group v-model='ruleForm.status' :disabled='disabledPhone'>
					<el-radio label='2'><%=rb.getString("YouXiaoQiSuoDing")%></el-radio>
					<el-radio label='0' style='margin-left:150px;'><%=rb.getString("JieSuo")%></el-radio>
				</el-radio-group>
			</div>
		</el-form-item>
		<el-form-item class='basicItem' label='<%=rb.getString("MiMa")%>' prop='password' v-if='"${type}" != "modify"'>
			<el-password v-model="ruleForm.password" size="mini" placeholder="" show-password></el-password>
			<el-input v-model='ruleForm.password' style="display:none;"></el-input>
		</el-form-item>
		<el-form-item class='basicItem' label='<%=rb.getString("QueRenMiMa")%>' prop='confirmPwd' v-if='"${type}" != "modify"'>
			<el-password v-model="ruleForm.confirmPwd" size="mini" placeholder="" show-password></el-password>
			<el-input v-model='ruleForm.confirmPwd' style="display:none;"></el-input>
		</el-form-item>
		<div class='groupItem'>
			<label class='groupLabel'><%=rb.getString("YongHuZu")%></label>
			<div class="groupHide"></div>
			<div class='groupDiv'>
				<div style="flex: auto;padding:8px 0px 0px 0px" >
					<p class='leftItem' v-for='item in selectGroup' :key='item.id' :disabled='disabledPhone'>
						<span>{{item.name}}</span>
						<span class='el-icon el-icon-circle-close' @click='deleteGroup(item)' style='font-size:14px;margin-left:15px;vertical-align:text-bottom'></span>
					</p>
				</div>
				<div @click='showflag = !showflag' style='display:flex;align-items:center;margin-right:10px;cursor:pointer'>
					<span class='el-icon el-icon-down' style='font-size:14px;'></span>
				</div>
			</div>
			<el-form-item prop='groupIds'>
				<el-input v-model='ruleForm.groupIds' v-show='false'></el-input>
			</el-form-item>
			<div class='selectGroupItem'>
				<transition name='el-zoom-in-top'>
					<div v-show='showflag' style='width:655px;height:auto;border:1px solid #DEDFE6;margin-top:5px;padding:20px;background:#fff'>
						<div style='padding-bottom:20px;padding-left:30px;'>
							<el-radio v-model='ruleForm.group' label='admin'>Super Admin</el-radio>
							<el-radio v-model='ruleForm.group' label='default' style='margin-left:100px;'>Default Group</el-radio>
						</div>
						<div v-if='showCustomize' style='display:flex;flex-direction:row;padding-left:30px;padding-top:20px;border-top:1px solid #F3F3F3;'>
							<div style='width:150px;'>
								<el-radio v-model='ruleForm.group' label='customize'>Customize</el-radio>
							</div>
							<div style='flex:1;'>
								<el-checkbox-group :disabled='disableGroup' v-model='checkList'>
									<el-checkbox v-for='item in checkGroup' :label='item.id+","+item.name' style='min-width:200px;margin-bottom:20px;'>{{item.name}}</el-checkbox>
								</el-checkbox-group>
							</div>
						</div>
						<div class="windowButtonGroup" style='float:left;margin-top:25px;'>
  							<a class="linkbutton linkbutton_trend" @click="saveGroup"><span><%=rb.getString("QueDing")%></span></a>
  							<a class="linkbutton linkbutton_nowanna" @click="cancelGroup()"><span><%=rb.getString("QuXiao")%></span></a>
						</div>
					</div>
				</transition>
			</div>
			<el-form-item prop='group' v-if='false'>
				<el-input v-model='ruleForm.group'></el-input>
			</el-form-item>
		</div>
		<div class='dateItem'>
			<p>
				<label class='dateLabel'><%=rb.getString("DaoQiShiJian")%></label>
			</p>
			<el-form-item style='margin-top:5px;' prop='expireTime'>
				<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" :disabled='disableTime' type='datetime' v-model='ruleForm.expireTime' :picker-options="pickerOptions"></el-date-picker>
				<el-checkbox v-model='ruleForm.limitTime' :disabled='disabledPhone' style='margin-left:25px;'><%=rb.getString("BuXianZhiShiJian")%></el-checkbox>
			</el-form-item>
			<el-form-item v-if='false'>
				<el-input v-model='ruleForm.limitTime'></el-input>
			</el-form-item>
		</div>
		<el-form-item label='<%=rb.getString("MiaoShu")%>' style='margin-top:20px;' prop='desc'>
			<el-input type='textarea' :rows='4' v-model='ruleForm.desc' :disabled='disabledPhone' maxlength="500"></el-input>
		</el-form-item>
	</el-form>
</div>
<script>
	var phoneShow = '';
	var addVue = new Vue({
		el:'#addUserDiv',
		data(){
			//校验用户名
			var validateUserName = (rule,value,callback) => {
				if ("${type}" == "modify"){
					callback();
					return;
				}
				var vm = this;
				var chineseReg = /.*[\u4e00-\u9fa5]+.*$/;
				if(chineseReg.test(value)){
					callback(new Error('<%=rb.getString("YongHuMingBuNengHanYouZhongWen")%>'))
				}

				axios.post('${ctx}/system/sysuser/checkUserCodeEnable.action').then(function(res){
					var data = res.data,
						needCheck = data == true,
						onlyInReg = /^[a-zA-Z0-9_\-—]+$/,//只能包含数字、26个小写字母、大写字母和减号(-)、下划线(_)、破折号(—)
						threeReg = /^(?=.*[a-z])(?=.*[A-Z])(?=.*[-_——])[a-zA-z_—-]{3,}$/; // 至少保护大小写字母、减号(-)、下划线(_)、破折号(—)

					if(needCheck && !onlyInReg.test(value)) {
						callback('<%=rb.getString("YongHuMingBiHanZiFuTiShi")%>');
					}

					axios.post("${ctx}/system/sysuser/queryPasswordRule.action").then(function(response){
						var data = response.data;
						var min_username = data.min_username;
						var max_username = 50;
						if(value.length > 50 || value.length < min_username){
							var msg = "<%=rb.getString("ZiFuChang")%>";
							//如果是英文浏览器添加一个空格
							if ("${i18n_type}" != "zh") {
								msg += " ";
							}
							msg += "<%=rb.getString("MaoHao")%>"
							+ min_username + '-' + max_username;
							callback(new Error(msg))
						}else{
							if(vm.defaultUserName == value){
								callback();
							}else{
								var params = {
										user_code : value
								}
								axios.post('${ctx}/system/sysuser/findUserExist.action',stringify(params)).then(function(response){
									var data = response.data;
									if(data["success"]){
										callback(new Error('<%=rb.getString("Msg_DengLuYongHuMingYiCunZai")%>'))
									}else{
										callback();
									}
								})
							}
						}
					})
				});
			}
			//校验邮箱格式
			var validateEmail = (rule,value,callback) => {
				if(value == '' || value == null){
					callback();
					return;
				}
				if(value.length > 50){
					var msg = "<%=rb.getString("ChangDuChaoChuFanWei")%><%=rb.getString("DouHao")%>";
					msg += "<%=rb.getString("ZuiDaChangDu")%>";
					msg += "<%=rb.getString("MaoHao")%> ";
					msg += 50;
					msg += " <%=rb.getString("ZiFu")%>";
					callback(new Error(msg))
				}else{
					var reg = /^([a-zA-Z0-9_\.\-])+@([a-zA-Z0-9_-])+(\.[a-zA-Z0-9_-]+)+$/;
					if(reg.test(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("YouXiangGeShiCuoWu")%>'))
					}
				}
			}
			//校验电话号码格式
			var validatePhone = (rule,value,callback) => {
				if (phoneShow == "false") {
					if(value.length > 20){
						var msg = "<%=rb.getString("ChangDuChaoChuFanWei")%><%=rb.getString("DouHao")%>";
						msg += "<%=rb.getString("ZuiDaChangDu")%>";
						msg +=20;
						msg += " <%=rb.getString("ZiFu")%>";
						callback(new Error(msg))
					}else{
						//固话校验
						var reg = /^[0-9-+() ]*$/;
						if(reg.test(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("DianHuaGeShi")%>'))
						} 
					} 
				} else {						
					//手机号校验
					var phoneReg = /^1\d{10}$/;
					if(phoneReg.test(value)){
						callback();
					}else if(value == ''){
						callback(new Error('<%=rb.getString("QingShuRuShouJiHao")%>'));
					}else{
						callback(new Error('<%=rb.getString("ShouJiHaoGeShi")%>'))
					}	 			
				}
			}
			//校验密码
			var validatePwd = (rule,value,callback) => {
				var vm = this;

				axios.post('${ctx}/system/sysuser/getPwdLength.action').then(function(res){
					var data = res.data,
						minLength = data.minLength,
						maxLength = data.maxLength,
						reg = /^[a-zA-Z0-9_!@#$%^&*?]+$/;
					
					var msg = "<%=rb.getString("MiMaChangDu")%> " + minLength + "-" + maxLength+ " <%=rb.getString("ZiFuFuShu")%>" 
								+ ", <%=rb.getString("FanWei")%>: [a-zA-Z0-9]、<%=rb.getString("TeSHuFuHao")%>(_!@#$%^&*?)";
					
					/*if (value.length < parseInt(minLength) || value.length > parseInt(maxLength) || !reg.test(value)) {
						callback(new Error(msg));
					}*/
					
					axios.post('${ctx}/system/sysuser/queryPasswordRule.action').then(function(response){
						var data = response.data;
						var passwordContent = data.password_content;
						/*
						var minLength = 6;
						var maxLength = 20;
						var reg = /^[a-zA-Z0-9_!@#$%^&*?]{6,20}$/;
						var msg = "<%=rb.getString("MiMaChangDu")%> " + minLength + "-" + maxLength+ " <%=rb.getString("ZiFuFuShu")%>" + ", <%=rb.getString("FanWei")%>: [a-zA-Z0-9]、<%=rb.getString("TeSHuFuHao")%>(_!@#$%^&*?)";
						if (!(value.length >= parseInt(minLength) && value.length <= parseInt(maxLength)) || !reg.test(value)) {
							callback(new Error(msg));
						}
						*/
						if (passwordContent == "0") {
							// 密码组成规则有限制
							if (value.length < parseInt(minLength) || value.length > parseInt(maxLength) || checkPasswdStrength(value) == "1") {
								var msgTip = '<%=rb.getString("MiMaChangDu")%> ' + minLength + "-" + maxLength+ ' <%=rb.getString("ZiFuFuShu")%>' 
								+ ', <%=rb.getString("MiMaBiXuLiangZhongLeiXing")%>';
								callback(new Error(msgTip))
							} else {
								// 密码OK
								callback();
								if(vm.ruleForm.confirmPwd != ''){
									vm.$refs.ruleForm.validateField('confirmPwd');
								}
							}
						} else {
							if (value.length < parseInt(minLength) || value.length > parseInt(maxLength) || !reg.test(value)) {
								callback(new Error(msg));
							}else{
								callback();
								if(vm.ruleForm.confirmPwd != ''){
									vm.$refs.ruleForm.validateField('confirmPwd');
								}
							}
							/*callback();
							if(vm.ruleForm.confirmPwd != ''){
								vm.$refs.ruleForm.validateField('confirmPwd');
							}*/
						}
					})
				})

			}
			//校验确认密码
			var validateConfirmPwd = (rule,value,callback) => {
				if(this.ruleForm.confirmPwd == this.ruleForm.password){
					callback();
				}else{
					callback(new Error('<%=rb.getString("LiangMiMaBuYiZhi")%>'))
				}
			}
			//校验时间
			var validateTime = (rule,value,callback) => {
				if(this.ruleForm.limitTime){//不限制
					callback()
				}else{
					if(value == ''||value==null){
						callback(new Error('<%=rb.getString("QingXuanZeDaoQiShiJian")%>'))
					}else{
						if(new Date(value).getTime() - Date.now() < 0) {
							callback('<%=rb.getString("DaYuDangQianShiJian")%>')
						}else {
							callback();
						}
					}
				}
			}
			return{
				ruleForm:{
					userName:'',
					email:'',
					phone:'',
					status:'0',
					password:'',
					confirmPwd:'',
					group:'default',
					expireTime:'',
					limitTime:true,
					desc:'',
					groupIds:'100000'
				},
				rules:{
					userName:[
						{required:true,message:'<%=rb.getString("QingShuRuYongHuMing")%>',trigger:'blur'},
						{validator:validateUserName,trigger:'blur'}
					],
					email:[
						{validator:validateEmail,trigger:'blur'}
					],
					phone:[
						{validator:validatePhone,trigger:'blur'}
					],
					password:[
						{required:true,message:'<%=rb.getString("QingShuRuMiMa")%>'},
						{validator:validatePwd}
					],
					confirmPwd:[
						{required:true,message:'<%=rb.getString("QingShuRuQueRenMiMa")%>'},
						{validator:validateConfirmPwd}
					],
					groupIds:[
						{required:true,message:'<%=rb.getString("QingXuanZeYongYuZu")%>',trigger:'change'}
					],
					expireTime:[
						{type:'date',validator:validateTime,trigger:'change'}
					]
				},
				selectGroup:[
					{
						name:'Default Group',
						id:100000
					}
				],
				showflag:false,
				radio:'2',
				checkGroup:[],
				expiresDate:'30',
				disableGroup:true,
				checkList:[],
				disabledPhone:true, 
				disableTime:true,
				showCustomize:false,
				defaultUserName:'',
				pickerOptions:{
					disabledDate(time){
						return time.getTime()< Date.now()-8.64e7;
					}
				}
			}
		},
		methods:{
			init(){
				var dateStr = dateformatter(new Date()).replace(' ','-').substr(3);
				//需判断是否是短信验证模式
				$.post("${ctx}/system/sysuser/msgmod.action", {rd: AesEncrypt(dateStr)}, function(data){
		        	var result = AesDecrypt(data.result||'4yNThc1APBpYgFK4s6OvOw==', CryptoJS.enc.Utf8.parse(dateStr));
		        	
					phoneShow = result.msgEnable4Num;
					if (result.msgEnable4Num == "false") {
						$('.phoneRequired').hide();
					} else {
						$('.phoneRequired').show();
					}  						
				}, "json");
				
				var vm = this;
				vm.disabledPhone = false; //可点击
				/* vm.disableGroup = false; */
				$('.groupHide').hide();		
				axios.post("${ctx}/sys/usergroup/getUserGroupForCombobox.action").then(function(response){
					var data = response.data;
					if(data.length > 0){
						vm.showCustomize = true;
						data.map(function(item){
							vm.checkGroup.push({
								name:item.text,
								id:item.id,
								check:false
							})
						})
					}else{
						vm.showCustomize = false;
					}
				})
				if("${type}" != 'add'){
					if("${type}" == 'modify'){
						axios.post("${ctx}/system/sysuser/getUserDetil.action",stringify({user_id:userVue.userRowData.id,timeZone:timeZone})).then(function(response){
							var data = response.data;
							vm.ruleForm.userName = data.user_code;
							vm.defaultUserName = data.user_code;
							vm.ruleForm.email = data.user_email;
							vm.ruleForm.phone = data.user_cell ? data.user_cell : '';
							vm.ruleForm.desc = data.user_desc;
							vm.ruleForm.status = data.lock_status;
							if(data.expites_date == ""){
								vm.ruleForm.limitTime = true;
							}else{
								vm.ruleForm.expireTime = data.expites_date;
								vm.ruleForm.limitTime = false;
							}
							
							if(data.build_in == "1"){ // 账号为 admin，只允许修改手机号
								vm.disabledPhone = true;
								vm.disableGroup = true;
								$('.groupHide').show();
							}
						})
					}
					axios.post("${ctx}/system/sysuser/getUserGroupSelected.action",stringify({user_id:userVue.userRowData.id})).then(function(response){
						var data = response.data;
						var dataArr = [];
						var checkArr = [];
						data.map(function(item){
							
							if(item.user_group_id == 1){
								vm.ruleForm.group = 'admin';
							}else if(item.user_group_id == 100000){
								vm.ruleForm.group = 'default';
							}else{
								vm.ruleForm.group = 'customize';
								checkArr.push(item.user_group_id+","+item.group_name);
							}
							dataArr.push({
								name:item.group_name,
								id:item.user_group_id
							})
							
						})
						vm.checkList = checkArr;
						vm.selectGroup = dataArr;
					})
					setTimeout(function(){
						initForm(vm.$refs.ruleForm);
					},100)
				}
			},
			/**
			 * 取消用户组勾选方法
			 * param item {object} 删除项
			*/
			deleteGroup(item){
				var vm = this;
				var index = vm.selectGroup.indexOf(item);
				if(index != -1){
					vm.selectGroup.splice(index,1);
				}
				if(item.id == 1 || item.id == 100000){
					vm.ruleForm.group = ''
				}else{
					var ids = vm.checkList.map(function(item,index){
						return item.split(',')[0];
					})
					var index = ids.indexOf(item.id);
					vm.checkList.splice(index,1);
				}
			},
			//保存勾选用户组方法
			saveGroup(){
				var vm = this;
				if(vm.ruleForm.group == 'admin'){
					vm.selectGroup = [{name:'Super Admin',id:1}]
				}else if(vm.ruleForm.group == 'default'){
					vm.selectGroup = [{name:'Default Group',id:100000}]
				}else if(vm.ruleForm.group == 'customize'){
					vm.selectGroup = [];
					vm.checkList.map(function(item){
						var list = item.split(',');
						vm.selectGroup.push({
							name:list[1],
							id:list[0]
						})
					})
				}
				vm.showflag = false
			},
			//关闭选择用户组下拉框
			cancelGroup(){
				this.showflag = false;
			},
			//新建/修改用户提交方法
			submit(){
				var vm = this;
				var tipStr = '<%=rb.getString("WuCanShuBianHua")%>';
				vm.$refs.ruleForm.validate((valid) => {
					if(valid){
						if(isFormChanged(vm.$refs.ruleForm)){
							if("${type}" == 'add' || "${type}" == 'copy'){
								var params = {
										user_code: RsaEncrypt(vm.ruleForm.userName, '${publicKey}'),
										user_pwd: RsaEncrypt(vm.ruleForm.password, '${publicKey}'),
										group_id:vm.ruleForm.groupIds,
										user_cell:vm.ruleForm.phone,
										user_email:vm.ruleForm.email,
										user_desc:vm.ruleForm.desc,
										lock_status:vm.ruleForm.status,
										timeZone:timeZone
								}
								if("${type}" == 'copy'){
									var message = '<%=rb.getString("ChengGong")%>';
								}else{
									var message = '<%=rb.getString("ChengGong")%>';
								}
								var url = "${ctx}/system/sysuser/addUser.action";
							}else if("${type}" == 'modify'){
								var params = {
										user_code: RsaEncrypt(vm.ruleForm.userName, '${publicKey}'),
										group_id:vm.ruleForm.groupIds,
										user_cell:vm.ruleForm.phone,
										user_email:vm.ruleForm.email,
										user_desc:vm.ruleForm.desc,
										lock_status:vm.ruleForm.status,
										timeZone:timeZone,
										user_id:userVue.userRowData.id
								}
								var message = '<%=rb.getString("ChengGong")%>';
								var url = "${ctx}/system/sysuser/doUpdate.action"
							}
							if(vm.ruleForm.limitTime){
								params.expites_date = 'null'
							}else{
								params.expites_date = vm.ruleForm.expireTime
							}
							
							axios.post(url,stringify(params)).then(function(response){
								var data = response.data;
								if(data["success"]){
									vm.$message({
			    						message:message,
			    						type:'success',
			    					})
                                    userVue.$refs.userTable.refresh();
                                    userVue.$refs.slide.hide();
                                    if("${type}" == 'add'){
                                        userVue.showAddButton = true;
                                    }
								}else{
									vm.$message.error(data["message"])
								}
							})
						}else{
							vm.$message("<%=rb.getString("WuCanShuBianHua")%>")
						}
					}
				})
			},
			//关闭新建/修改用户页面方法
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
			},
		},
		watch:{
			selectGroup(){
				var vm = this;
				var groupIds = '';
				vm.selectGroup.map(function(item){
					groupIds += item.id+',';
				})
				vm.ruleForm.groupIds = groupIds.substring(0,groupIds.length-1);
			},
			"ruleForm.group":function(newVal){
				if(newVal == 'customize'){
					this.disableGroup = false
				}else{
					this.disableGroup = true;
				}
			},
			"ruleForm.limitTime":function(newVal){
				if(newVal){
					this.disableTime = true;
					this.ruleForm.expireTime = ''
				}else{
					this.disableTime = false;
				}
			}
		},
		mounted(){
			this.init();
			eventBus.$off('save-user').$on('save-user',this.submit);
			eventBus.$off('cancel-user').$on('cancel-user',this.cancel);
		}
	})
</script>